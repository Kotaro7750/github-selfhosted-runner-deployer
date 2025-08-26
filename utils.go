package main

import (
	"github.com/google/uuid"
)

func generateRunnerID() string {
	for {
		id := uuid.NewString()

		_, exist := runners[id]
		if !exist {
			return id
		}
	}
}
