package main

import (
	"os"
	"testing"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core/coretest"
)

func TestMain(m *testing.M) {
	os.Exit(coretest.Run(m))
}
