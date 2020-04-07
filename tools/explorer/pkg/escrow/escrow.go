package escrow

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/config"
	gdirectory "github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/explorer/pkg/directory"
	directorytypes "github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/explorer/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"
	workloadtypes "github.com/threefoldtech/zos/tools/explorer/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// Escrow service manages a dedicate wallet for payments for reservations.
	Escrow struct {
		wallet *stellar.Wallet
		db     *mongo.Database

		reservationChannel chan reservationRegisterJob
		deployedChannel    chan schema.ID
		cancelledChannel   chan schema.ID

		nodeAPI NodeAPI
		farmAPI FarmAPI

		ctx context.Context
	}

	// NodeAPI operations on node database
	NodeAPI interface {
		// Get a node from the database using its ID
		Get(ctx context.Context, db *mongo.Database, id string, proofs bool) (directorytypes.Node, error)
	}

	// FarmAPI operations on farm database
	FarmAPI interface {
		// GetByID get a farm from the database using its ID
		GetByID(ctx context.Context, db *mongo.Database, id int64) (directorytypes.Farm, error)
	}

	reservationRegisterJob struct {
		reservation  workloads.Reservation
		responseChan chan reservationRegisterJobResponse
	}

	reservationRegisterJobResponse struct {
		data types.CustomerEscrowInformation
		err  error
	}
)

const (
	// interval between every check of active escrow accounts
	balanceCheckInterval = time.Minute * 1
)

// New creates a new escrow object and fetches all addresses for the escrow wallet
func New(wallet *stellar.Wallet, db *mongo.Database) *Escrow {
	jobChannel := make(chan reservationRegisterJob)
	deployChannel := make(chan schema.ID)
	cancelChannel := make(chan schema.ID)

	return &Escrow{
		wallet:             wallet,
		db:                 db,
		nodeAPI:            &directory.NodeAPI{},
		farmAPI:            &directory.FarmAPI{},
		reservationChannel: jobChannel,
		deployedChannel:    deployChannel,
		cancelledChannel:   cancelChannel,
	}
}

// Run the escrow until the context is done
func (e *Escrow) Run(ctx context.Context) error {
	ticker := time.NewTicker(balanceCheckInterval)
	defer ticker.Stop()

	e.ctx = ctx

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("escrow context done, exiting")
			return nil

		case <-ticker.C:
			log.Info().Msg("scanning active escrow accounts balance")
			if err := e.checkReservations(); err != nil {
				log.Error().Err(err).Msgf("failed to check reservations")
			}

			log.Info().Msg("scanning for expired escrows")
			if err := e.refundExpiredReservations(); err != nil {
				log.Error().Err(err).Msgf("failed to refund expired reservations")
			}

		case job := <-e.reservationChannel:
			log.Info().Int64("reservation_id", int64(job.reservation.ID)).Msg("processing new reservation escrow for reservation")
			details, err := e.processReservation(job.reservation)
			if err != nil {
				log.Error().
					Err(err).
					Int64("reservation_id", int64(job.reservation.ID)).
					Msgf("failed to check reservations")
			}
			job.responseChan <- reservationRegisterJobResponse{
				err:  err,
				data: details,
			}

		case id := <-e.deployedChannel:
			log.Info().Int64("reservation_id", int64(id)).Msg("trying to pay farmer for deployed reservation")
			if err := e.payoutFarmers(id); err != nil {
				log.Error().
					Err(err).
					Int64("reservation_id", int64(id)).
					Msgf("failed to payout farmers")
			}

		case id := <-e.cancelledChannel:
			log.Info().Int64("reservation_id", int64(id)).Msg("trying to refund clients for canceled reservation")
			if err := e.refundClients(id); err != nil {
				log.Error().
					Err(err).
					Int64("reservation_id", int64(id)).
					Msgf("could not refund clients")
			}
		}
	}
}

func (e *Escrow) refundExpiredReservations() error {
	// load expired escrows
	reservationEscrows, err := types.GetAllExpiredReservationPaymentInfos(e.ctx, e.db)
	if err != nil {
		return errors.Wrap(err, "failed to load active reservations from escrow")
	}
	for _, escrowInfo := range reservationEscrows {
		log.Info().Int64("id", int64(escrowInfo.ReservationID)).Msg("expired escrow")

		if err := e.refundEscrow(escrowInfo); err != nil {
			log.Error().Err(err).Msgf("failed to refund reservation escrow")
			continue
		}

		escrowInfo.Canceled = true
		if err = types.ReservationPaymentInfoUpdate(e.ctx, e.db, escrowInfo); err != nil {
			log.Error().Err(err).Msgf("failed to mark expired reservation escrow info as cancelled")
		}
	}
	return nil
}

// checkReservations checks all the active reservations and marks those who are funded.
// if a reservation is funded then it will mark this reservation as to DEPLOY.
// if its underfunded it will throw an error.
func (e *Escrow) checkReservations() error {
	// load active escrows
	reservationEscrows, err := types.GetAllActiveReservationPaymentInfos(e.ctx, e.db)
	if err != nil {
		return errors.Wrap(err, "failed to load active reservations from escrow")
	}

	for _, escrowInfo := range reservationEscrows {
		if err := e.checkReservationPaid(escrowInfo); err != nil {
			log.Error().
				Str("address", escrowInfo.Address).
				Int64("reservation_id", int64(escrowInfo.ReservationID)).
				Err(err).
				Msg("failed to check reservation escrow funding status")
			continue
		}
	}
	return nil
}

// CheckReservationPaid verifies if an escrow account received sufficient balance
// to pay for a reservation. If this is the case, the reservation will be moved
// to the deploy state, and the escrow state will be updated to indicate that this
// escrow has indeed been paid for this reservation, so it is not checked anymore
// in the future.
func (e *Escrow) checkReservationPaid(escrowInfo types.ReservationPaymentInformation) error {
	slog := log.With().
		Str("address", escrowInfo.Address).
		Int64("reservation_id", int64(escrowInfo.ReservationID)).
		Logger()

	// calculate total amount needed for reservation
	requiredValue := xdr.Int64(0)
	for _, escrowAccount := range escrowInfo.Infos {
		requiredValue += escrowAccount.TotalAmount
	}

	balance, _, err := e.wallet.GetBalance(escrowInfo.Address, escrowInfo.ReservationID)
	if err != nil {
		return errors.Wrap(err, "failed to verify escrow account balance")
	}

	if balance < requiredValue {
		slog.Debug().Msgf("required balance %d not reached yet (%d)", requiredValue, balance)
		return nil
	}

	slog.Debug().Msgf("required balance %d funded (%d), continue reservation", requiredValue, balance)

	reservation, err := workloadtypes.ReservationFilter{}.WithID(escrowInfo.ReservationID).Get(e.ctx, e.db)
	if err != nil {
		return errors.Wrap(err, "failed to load reservation")
	}

	pl, err := workloadtypes.NewPipeline(reservation)
	if err != nil {
		return errors.Wrap(err, "failed to process reservation pipeline")
	}

	reservation, _ = pl.Next()
	if !reservation.IsAny(workloadtypes.Pay) {
		// Do not continue, but also take no action to drive the reservation
		// as much as possible from the main explorer part.
		slog.Warn().Msg("reservation is paid, but no longer in pay state")
		// We warn because this is an unusual state to be in, yet there are
		// situations where this could happen. For example, we load the escrow,
		// the explorer then invalidates the actual reservation (e.g. user cancels),
		// we then load the updated reservation, which is no longer in pay state,
		// but the explorer is still cancelling the escrow, so we get here. As stated
		// above, we drive the escrow as much as possible from the workloads, with the
		// timeouts coming from the escrow itself, so this situation should always
		// resole itself. If we notice this log is coming back periodically, it thus means
		// there is a bug somewhere else in the code.
		// As a result, this state is therefore not considered an error.
		return nil
	}

	slog.Info().Msg("all farmer are paid, trying to move to deploy state")

	// update reservation
	if err = workloadtypes.ReservationSetNextAction(e.ctx, e.db, escrowInfo.ReservationID, workloadtypes.Deploy); err != nil {
		return errors.Wrap(err, "failed to set reservation to DEPLOY state")
	}

	escrowInfo.Paid = true
	if err = types.ReservationPaymentInfoUpdate(e.ctx, e.db, escrowInfo); err != nil {
		return errors.Wrap(err, "failed to mark reservation escrow info as paid")
	}

	slog.Debug().Msg("escrow marked as paid")

	return nil
}

// processReservation processes a single reservation
// calculates resources and their costs
func (e *Escrow) processReservation(reservation workloads.Reservation) (types.CustomerEscrowInformation, error) {
	var customerInfo types.CustomerEscrowInformation
	rsuPerFarmer, err := e.processReservationResources(reservation.DataReservation)
	if err != nil {
		return customerInfo, errors.Wrap(err, "failed to process reservation resources")
	}

	res, err := e.calculateReservationCost(rsuPerFarmer)
	if err != nil {
		return customerInfo, errors.Wrap(err, "failed to process reservation resources costs")
	}

	address, err := e.createOrLoadAccount(reservation.CustomerTid)
	if err != nil {
		return customerInfo, errors.Wrap(err, "failed to get escrow address for customer")
	}

	details := make([]types.EscrowDetail, 0, len(res))
	for farmer, value := range res {
		if err != nil {
			return customerInfo, errors.Wrap(err, "failed to create or load account")
		}
		details = append(details, types.EscrowDetail{
			FarmerID:    schema.ID(farmer),
			TotalAmount: value,
		})
	}
	reservationPaymentInfo := types.ReservationPaymentInformation{
		Infos:         details,
		Address:       address,
		ReservationID: reservation.ID,
		Expiration:    reservation.DataReservation.ExpirationProvisioning,
		Paid:          false,
		Canceled:      false,
		Released:      false,
	}
	err = types.ReservationPaymentInfoCreate(e.ctx, e.db, reservationPaymentInfo)
	if err != nil {
		return customerInfo, errors.Wrap(err, "failed to create reservation payment information")
	}
	log.Info().Int64("id", int64(reservation.ID)).Msg("processed reservation and created payment information")
	customerInfo.Address = address
	customerInfo.Details = details
	return customerInfo, nil
}

// refundClients refunds clients if the reservation is cancelled
func (e *Escrow) refundClients(id schema.ID) error {
	rpi, err := types.ReservationPaymentInfoGet(e.ctx, e.db, id)
	if err != nil {
		return errors.Wrap(err, "failed to get reservation escrow info")
	}
	if rpi.Released || rpi.Canceled {
		// already paid
		return nil
	}
	if err := e.refundEscrow(rpi); err != nil {
		log.Error().Err(err).Msg("failed to refund escrow")
		return errors.Wrap(err, "could not refund escrow")
	}
	rpi.Canceled = true
	if err = types.ReservationPaymentInfoUpdate(e.ctx, e.db, rpi); err != nil {
		return errors.Wrapf(err, "could not mark escrows for %d as canceled", rpi.ReservationID)
	}
	log.Debug().Int64("id", int64(rpi.ReservationID)).Msg("refunded clients for reservation")
	return nil
}

// payoutFarmers pays out the farmer for a processed reservation
func (e *Escrow) payoutFarmers(id schema.ID) error {
	rpi, err := types.ReservationPaymentInfoGet(e.ctx, e.db, id)
	if err != nil {
		return errors.Wrap(err, "failed to get reservation escrow info")
	}
	if rpi.Released || rpi.Canceled {
		// already paid
		return nil
	}

	// collect the farmer addresses and amount they should receive, we already
	// have sufficient balance on the escrow to cover this
	paymentInfo := make([]stellar.PayoutInfo, 0, len(rpi.Infos))
	for _, escrowDetails := range rpi.Infos {
		// in case of an error in this flow we continue, so we try to pay as much
		// farmers as possible even if one fails
		farm, err := e.farmAPI.GetByID(e.ctx, e.db, int64(escrowDetails.FarmerID))
		if err != nil {
			log.Error().Msgf("failed to load farm info: %s", err)
			continue
		}

		destination, err := addressByAsset(farm.WalletAddresses, config.Config.Asset)
		if err != nil {
			// FIXME: this is probably not ok, what do we do in this case ?
			log.Error().Err(err).Msgf("failed to find address for %s for farmer %d", config.Config.Asset, farm.ID)
			continue
		}

		paymentInfo = append(paymentInfo,
			stellar.PayoutInfo{
				Address: destination,
				Amount:  escrowDetails.TotalAmount,
			},
		)
	}

	addressInfo, err := types.CustomerAddressByAddress(e.ctx, e.db, rpi.Address)
	if err != nil {
		log.Error().Msgf("failed to load escrow address info: %s", err)
		return errors.Wrap(err, "could not load escrow address info")
	}
	kp, err := e.wallet.KeyPairFromSeed(addressInfo.Secret)
	if err != nil {
		log.Error().Msgf("failed to parse escrow address secret: %s", err)
		return errors.Wrap(err, "could not load escrow address info")
	}
	if err = e.wallet.PayoutFarmers(*kp, paymentInfo, id); err != nil {
		log.Error().Msgf("failed to pay farmer: %s for reservation %d", err, id)
		return errors.Wrap(err, "could not pay farmer")
	}
	// now refund any possible overpayment
	if err = e.wallet.Refund(*kp, id); err != nil {
		log.Error().Msgf("failed to refund overpayment farmer: %s", err)
		return errors.Wrap(err, "could not refund overpayment")
	}
	log.Info().
		Str("escrow address", rpi.Address).
		Int64("reservation id", int64(rpi.ReservationID)).
		Msgf("paid farmer")

	rpi.Released = true
	if err = types.ReservationPaymentInfoUpdate(e.ctx, e.db, rpi); err != nil {
		return errors.Wrapf(err, "could not mark escrows for %d as released", rpi.ReservationID)
	}
	return nil
}

func (e *Escrow) refundEscrow(escrowInfo types.ReservationPaymentInformation) error {
	slog := log.With().
		Str("address", escrowInfo.Address).
		Int64("reservation_id", int64(escrowInfo.ReservationID)).
		Logger()

	slog.Info().Msgf("try to refund client for escrow")

	addressInfo, err := types.CustomerAddressByAddress(e.ctx, e.db, escrowInfo.Address)
	if err != nil {
		return errors.Wrap(err, "failed to load escrow info")
	}

	kp, err := e.wallet.KeyPairFromSeed(addressInfo.Secret)
	if err != nil {
		return errors.Wrap(err, "failed to parse escrow address secret")
	}

	if err = e.wallet.Refund(*kp, escrowInfo.ReservationID); err != nil {
		return errors.Wrap(err, "failed to refund clients")
	}

	slog.Info().Msgf("refunded client for escrow")
	return nil
}

// RegisterReservation registers a workload reservation
func (e *Escrow) RegisterReservation(reservation workloads.Reservation) (types.CustomerEscrowInformation, error) {
	job := reservationRegisterJob{
		reservation:  reservation,
		responseChan: make(chan reservationRegisterJobResponse),
	}
	e.reservationChannel <- job

	response := <-job.responseChan

	return response.data, response.err
}

// ReservationDeployed informs the escrow that a reservation has been successfully
// deployed, so the escrow can release the funds to the farmer (and refund any excess)
func (e *Escrow) ReservationDeployed(reservationID schema.ID) {
	e.deployedChannel <- reservationID
}

// ReservationCanceled informs the escrow of a canceled reservation so it can refund
// the user
func (e *Escrow) ReservationCanceled(reservationID schema.ID) {
	e.cancelledChannel <- reservationID
}

// createOrLoadAccount creates or loads account based on farmer - customer id
func (e *Escrow) createOrLoadAccount(customerTID int64) (string, error) {
	res, err := types.CustomerAddressGet(context.Background(), e.db, customerTID)
	if err != nil {
		if err == types.ErrAddressNotFound {
			keypair, err := e.wallet.CreateAccount()
			if err != nil {
				return "", errors.Wrapf(err, "failed to create a new account for customer %d", customerTID)
			}
			err = types.CustomerAddressCreate(context.Background(), e.db, types.CustomerAddress{
				CustomerTID: customerTID,
				Address:     keypair.Address(),
				Secret:      keypair.Seed(),
			})
			if err != nil {
				return "", errors.Wrapf(err, "failed to save a new account for customer %d", customerTID)
			}
			log.Debug().
				Int64("customer", int64(customerTID)).
				Str("address", keypair.Address()).
				Msgf("created new escrow address for farmer-customer")
			return keypair.Address(), nil
		}
		return "", errors.Wrap(err, "failed to get farmer - customer address")
	}
	log.Debug().
		Int64("customer", int64(customerTID)).
		Str("address", res.Address).
		Msgf("escrow address found for customer")

	return res.Address, nil
}

func addressByAsset(addrs []gdirectory.WalletAddress, asset string) (string, error) {
	for _, a := range addrs {
		if a.Asset == asset && a.Address != "" {
			return a.Address, nil
		}
	}
	return "", fmt.Errorf("not address found for asset %s", asset)
}
