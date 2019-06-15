package upgrade

type transactionner interface {
	Start() error
	Commit() error
	Rollback() error
}
