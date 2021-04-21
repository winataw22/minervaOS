package primitives

// func (p *Primitives) decryptSecret(ctx context.Context, user uint32, secret string, version int) (string, error) {
// 	if len(secret) == 0 {
// 		return "", nil
// 	}

// 	engine := provision.GetEngine(ctx)

// 	identity := stubs.NewIdentityManagerStub(p.zbus)

// 	bytes, err := hex.DecodeString(secret)
// 	if err != nil {
// 		return "", err
// 	}

// 	var (
// 		out []byte
// 	)
// 	// now only one version is supported
// 	switch version {
// 	default:
// 		var userPubKey ed25519.PublicKey
// 		userPubKey, err = engine.Twins().GetKey(user)
// 		if err != nil || userPubKey == nil {
// 			return "", fmt.Errorf("failed to retrieve user %d public key: %s", user, err)
// 		}
// 		out, err = identity.DecryptECDH(bytes, userPubKey)
// 	}

// 	return string(out), err
// }
