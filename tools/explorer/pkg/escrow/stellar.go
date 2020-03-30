package escrow

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/schema"
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
	// Stellar service manages a dedicate wallet for payments for reservations.
	Stellar struct {
		foundationAddress string
		wallet            *stellar.Wallet
		db                *mongo.Database

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
		reservation            workloads.Reservation
		supportedCurrencyCodes []string
		responseChan           chan reservationRegisterJobResponse
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

const (
	// amount of digits of precision a calculated reservation cost has, at worst
	costPrecision = 6
)

var (
	// ErrNoCurrencySupported indicates a reservation was offered but none of the currencies
	// the farmer wants to pay in are currently supported
	ErrNoCurrencySupported = errors.New("none of the offered currencies are currently supported")
	// ErrNoCurrencyShared indicates that none of the currencies offered in the reservation
	// is supported by all farmers used
	ErrNoCurrencyShared = errors.New("none of the provided currencies is supported by all farmers")
)

// NewStellar creates a new escrow object and fetches all addresses for the escrow wallet
func NewStellar(wallet *stellar.Wallet, db *mongo.Database, foundationAddress string) *Stellar {
	jobChannel := make(chan reservationRegisterJob)
	deployChannel := make(chan schema.ID)
	cancelChannel := make(chan schema.ID)

	addr := foundationAddress
	if addr == "" {
		addr = wallet.PublicAddress()
	}

	return &Stellar{
		wallet:             wallet,
		db:                 db,
		foundationAddress:  addr,
		nodeAPI:            &directory.NodeAPI{},
		farmAPI:            &directory.FarmAPI{},
		reservationChannel: jobChannel,
		deployedChannel:    deployChannel,
		cancelledChannel:   cancelChannel,
	}
}

// Run the escrow until the context is done
func (e *Stellar) Run(ctx context.Context) error {
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
			details, err := e.processReservation(job.reservation, job.supportedCurrencyCodes)
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

func (e *Stellar) refundExpiredReservations() error {
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
func (e *Stellar) checkReservations() error {
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
func (e *Stellar) checkReservationPaid(escrowInfo types.ReservationPaymentInformation) error {
	slog := log.With().
		Str("address", escrowInfo.Address).
		Int64("reservation_id", int64(escrowInfo.ReservationID)).
		Logger()

	// calculate total amount needed for reservation
	requiredValue := xdr.Int64(0)
	for _, escrowAccount := range escrowInfo.Infos {
		requiredValue += escrowAccount.TotalAmount
	}

	balance, _, err := e.wallet.GetBalance(escrowInfo.Address, escrowInfo.ReservationID, escrowInfo.Asset)
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
func (e *Stellar) processReservation(reservation workloads.Reservation, offeredCurrencyCodes []string) (types.CustomerEscrowInformation, error) {
	var customerInfo types.CustomerEscrowInformation

	// filter out unsupported currencies
	currencies := []stellar.Asset{}
	for _, offeredCurrency := range offeredCurrencyCodes {
		asset, err := e.wallet.AssetFromCode(offeredCurrency)
		if err != nil {
			if err == stellar.ErrAssetCodeNotSupported {
				continue
			}
			return customerInfo, err
		}
		// Sanity check
		if _, exists := assetDistributions[asset]; !exists {
			// no payout distribution info set, log error and treat as if the asset
			// is not supported
			log.Error().Msgf("asset %s supported by wallet but no payout distribution found in escrow", asset)
			continue
		}
		currencies = append(currencies, asset)
	}

	if len(currencies) == 0 {
		return customerInfo, ErrNoCurrencySupported
	}

	rsuPerFarmer, err := e.processReservationResources(reservation.DataReservation)
	if err != nil {
		return customerInfo, errors.Wrap(err, "failed to process reservation resources")
	}

	// check which currencies are accepted by all farmers
	// the farm ids have conveniently been provided when checking the used rsu
	farmIDs := make([]int64, 0, len(rsuPerFarmer))
	var asset stellar.Asset
	for _, currency := range currencies {
		// if the farmer does not receive anything in the first place, they always
		// all agree on this currency
		if assetDistributions[currency].farmer == 0 {
			asset = currency
			break
		}
		// check if all used farms have an address for this asset set up
		supported, err := e.checkAssetSupport(farmIDs, asset)
		if err != nil {
			return customerInfo, errors.Wrap(err, "could not verify asset support")
		}
		if supported {
			asset = currency
			break
		}
	}

	if asset == "" {
		return customerInfo, ErrNoCurrencyShared
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
		Asset:         asset,
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
	customerInfo.Asset = asset
	customerInfo.Details = details
	return customerInfo, nil
}

// refundClients refunds clients if the reservation is cancelled
func (e *Stellar) refundClients(id schema.ID) error {
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
func (e *Stellar) payoutFarmers(id schema.ID) error {
	rpi, err := types.ReservationPaymentInfoGet(e.ctx, e.db, id)
	if err != nil {
		return errors.Wrap(err, "failed to get reservation escrow info")
	}
	if rpi.Released || rpi.Canceled {
		// already paid
		return nil
	}

	paymentDistribution, exists := assetDistributions[rpi.Asset]
	if !exists {
		return fmt.Errorf("no payment distribution found for asset %s", rpi.Asset)
	}

	// keep track of total amount to burn and to send to foundation
	var toBurn, toFoundation xdr.Int64

	// collect the farmer addresses and amount they should receive, we already
	// have sufficient balance on the escrow to cover this
	paymentInfo := make([]stellar.PayoutInfo, 0, len(rpi.Infos))

	for _, escrowDetails := range rpi.Infos {
		farmerAmount, burnAmount, foundationAmount := e.splitPayout(escrowDetails.TotalAmount, paymentDistribution)
		toBurn += burnAmount
		toFoundation += foundationAmount

		if farmerAmount > 0 {
			// in case of an error in this flow we continue, so we try to pay as much
			// farmers as possible even if one fails
			farm, err := e.farmAPI.GetByID(e.ctx, e.db, int64(escrowDetails.FarmerID))
			if err != nil {
				log.Error().Msgf("failed to load farm info: %s", err)
				continue
			}

			destination, err := addressByAsset(farm.WalletAddresses, rpi.Asset)
			if err != nil {
				// FIXME: this is probably not ok, what do we do in this case ?
				log.Error().Err(err).Msgf("failed to find address for %s for farmer %d", rpi.Asset.Code(), farm.ID)
				continue
			}

			// farmerAmount can't be pooled so add an info immediately
			paymentInfo = append(paymentInfo,
				stellar.PayoutInfo{
					Address: destination,
					Amount:  farmerAmount,
				},
			)
		}
	}

	// a burn is a transfer of tokens back to the issuer
	if toBurn > 0 {
		paymentInfo = append(paymentInfo,
			stellar.PayoutInfo{
				Address: rpi.Asset.Issuer(),
				Amount:  toBurn,
			})
	}

	// ship remainder to the foundation
	if toFoundation > 0 {
		paymentInfo = append(paymentInfo,
			stellar.PayoutInfo{
				Address: e.foundationAddress,
				Amount:  toFoundation,
			})
	}

	addressInfo, err := types.CustomerAddressByAddress(e.ctx, e.db, rpi.Address)
	if err != nil {
		log.Error().Msgf("failed to load escrow address info: %s", err)
		return errors.Wrap(err, "could not load escrow address info")
	}
	if err = e.wallet.PayoutFarmers(addressInfo.Secret, paymentInfo, id, rpi.Asset); err != nil {
		log.Error().Msgf("failed to pay farmer: %s for reservation %d", err, id)
		return errors.Wrap(err, "could not pay farmer")
	}
	// now refund any possible overpayment
	if err = e.wallet.Refund(addressInfo.Secret, id, rpi.Asset); err != nil {
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

func (e *Stellar) refundEscrow(escrowInfo types.ReservationPaymentInformation) error {
	slog := log.With().
		Str("address", escrowInfo.Address).
		Int64("reservation_id", int64(escrowInfo.ReservationID)).
		Logger()

	slog.Info().Msgf("try to refund client for escrow")

	addressInfo, err := types.CustomerAddressByAddress(e.ctx, e.db, escrowInfo.Address)
	if err != nil {
		return errors.Wrap(err, "failed to load escrow info")
	}

	if err = e.wallet.Refund(addressInfo.Secret, escrowInfo.ReservationID, escrowInfo.Asset); err != nil {
		return errors.Wrap(err, "failed to refund clients")
	}

	slog.Info().Msgf("refunded client for escrow")
	return nil
}

// RegisterReservation registers a workload reservation
func (e *Stellar) RegisterReservation(reservation workloads.Reservation, supportedCurrencies []string) (types.CustomerEscrowInformation, error) {
	job := reservationRegisterJob{
		reservation:            reservation,
		supportedCurrencyCodes: supportedCurrencies,
		responseChan:           make(chan reservationRegisterJobResponse),
	}
	e.reservationChannel <- job

	response := <-job.responseChan

	return response.data, response.err
}

// ReservationDeployed informs the escrow that a reservation has been successfully
// deployed, so the escrow can release the funds to the farmer (and refund any excess)
func (e *Stellar) ReservationDeployed(reservationID schema.ID) {
	e.deployedChannel <- reservationID
}

// ReservationCanceled informs the escrow of a canceled reservation so it can refund
// the user
func (e *Stellar) ReservationCanceled(reservationID schema.ID) {
	e.cancelledChannel <- reservationID
}

// createOrLoadAccount creates or loads account based on  customer id
func (e *Stellar) createOrLoadAccount(customerTID int64) (string, error) {
	res, err := types.CustomerAddressGet(context.Background(), e.db, customerTID)
	if err != nil {
		if err == types.ErrAddressNotFound {
			seed, address, err := e.wallet.CreateAccount()
			if err != nil {
				return "", errors.Wrapf(err, "failed to create a new account for customer %d", customerTID)
			}
			err = types.CustomerAddressCreate(context.Background(), e.db, types.CustomerAddress{
				CustomerTID: customerTID,
				Address:     address,
				Secret:      seed,
			})
			if err != nil {
				return "", errors.Wrapf(err, "failed to save a new account for customer %d", customerTID)
			}
			log.Debug().
				Int64("customer", int64(customerTID)).
				Str("address", address).
				Msgf("created new escrow address for customer")
			return address, nil
		}
		return "", errors.Wrap(err, "failed to get customer address")
	}
	log.Debug().
		Int64("customer", int64(customerTID)).
		Str("address", res.Address).
		Msgf("escrow address found for customer")

	return res.Address, nil
}

// splitPayout to a farmer in the amount the farmer receives, the amount to be burned,
// and the amount the foundation receives
func (e *Stellar) splitPayout(totalAmount xdr.Int64, distribution payoutDistribution) (xdr.Int64, xdr.Int64, xdr.Int64) {
	// we can't just use big.Float for this calculation, since we need to verify
	// the rounding afterwards

	// calculate missing precision digits, to perform percentage division without
	// floating point operations
	requiredPrecision := 2 + costPrecision
	missingPrecision := requiredPrecision - e.wallet.PrecisionDigits()

	multiplier := int64(1)
	if missingPrecision > 0 {
		multiplier = int64(math.Pow10(missingPrecision))
	}

	amount := int64(totalAmount) * multiplier

	baseAmount := amount / 100
	farmerAmount := baseAmount * int64(distribution.farmer)
	burnAmount := baseAmount * int64(distribution.burned)
	foundationAmount := baseAmount * int64(distribution.foundation)

	// collect parts which will be missing in division, if any
	var change int64
	change += farmerAmount % multiplier
	change += burnAmount % multiplier
	change += foundationAmount % multiplier

	// change is now necessarily a multiple of multiplier
	change /= multiplier
	// we tracked all change which would be removed by the following integer
	// devisions
	farmerAmount /= multiplier
	burnAmount /= multiplier
	foundationAmount /= multiplier

	// give change to whichever gets funds anyway, in the following order:
	//  - farmer
	//  - burned
	//  - foundation
	if farmerAmount != 0 {
		farmerAmount += change
	} else if burnAmount != 0 {
		burnAmount += change
	} else if foundationAmount != 0 {
		foundationAmount += change
	}

	return xdr.Int64(farmerAmount), xdr.Int64(burnAmount), xdr.Int64(foundationAmount)
}

// checkAssetSupport for all unique farms in the reservation
func (e *Stellar) checkAssetSupport(farmIDs []int64, asset stellar.Asset) (bool, error) {
	for _, id := range farmIDs {
		farm, err := e.farmAPI.GetByID(e.ctx, e.db, id)
		if err != nil {
			return false, errors.Wrap(err, "could not load farm")
		}
		if _, err := addressByAsset(farm.WalletAddresses, asset); err != nil {
			// this only errors if the asset is not present
			return false, nil
		}
	}
	return true, nil
}

func addressByAsset(addrs []gdirectory.WalletAddress, asset stellar.Asset) (string, error) {
	for _, a := range addrs {
		if a.Asset == asset.Code() && a.Address != "" {
			return a.Address, nil
		}
	}
	return "", fmt.Errorf("not address found for asset %s", asset)
}
