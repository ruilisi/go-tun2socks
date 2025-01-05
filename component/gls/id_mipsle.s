// Copyright 2018 Massimiliano Ghilardi. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build gc

#include "go_asm.h"
#include "textflag.h" // for NOSPLIT
#include "go_tls.h"

TEXT ·getg(SB),NOSPLIT|NOFRAME,$-4-4
	MOVW g, ret+0(FP)
	RET
