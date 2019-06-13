package namespace

func ExampleNSContext() {
	// create a NSContext object
	nsCtx := NSContext{}
	// enter the nework namespace named nsName
	_ = nsCtx.Enter("nsNam")
	// exit the namespace and come back to the host namesapce
	defer nsCtx.Exit()
}
