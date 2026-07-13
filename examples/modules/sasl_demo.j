#!/usr/bin/env -S jennifer run
# SPDX-License-Identifier: LGPL-3.0-only
# Copyright (C) 2026 <developer@mplx.eu>

/**
 * The SASL authentication encoders (pure base64) that the mail clients (smtp / pop / imap) use to send credentials.
 * No networking; each call just renders the base64 token a server expects.
 * @module sasl_demo
 */
use io;
import "../../modules/sasl.j" as sasl;

# PLAIN: authzid \0 authcid \0 password, base64-encoded in one step.
io.printf("PLAIN       -> %s\n", sasl.plain("alice", "s3cret"));

# LOGIN: username and password sent as two separate base64 challenges.
io.printf("LOGIN user  -> %s\n", sasl.loginUser("alice"));
io.printf("LOGIN pass  -> %s\n", sasl.loginPass("s3cret"));

# XOAUTH2 (the "use a token" half of OAuth2): user + OAuth2 access token.
io.printf("XOAUTH2     -> %s\n", sasl.bearer("alice@example.com", "ya29.a0AccessToken"));
