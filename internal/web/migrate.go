package web

import (
	"log"
	"strings"

	"github.com/clawfleet/clawfleet/internal/container"
	"github.com/clawfleet/clawfleet/internal/state"
)

// runtimeFromImage classifies a container's image reference into a runtime
// type. Returns "hermes" if the ref contains the configured Hermes image
// name, "openclaw" otherwise.
func runtimeFromImage(imageRef, hermesImageName string) string {
	if hermesImageName == "" || imageRef == "" {
		return "openclaw"
	}
	if strings.Contains(strings.ToLower(imageRef), strings.ToLower(hermesImageName)) {
		return "hermes"
	}
	return "openclaw"
}

// imageLookupFunc resolves a container's image ref. A non-nil error tells the
// caller to skip that instance instead of misclassifying it.
type imageLookupFunc func(containerID string) (string, error)

// backfillRuntimeTypes fills in RuntimeType for instances whose state.json
// entry is missing the field — a one-shot self-heal for records written by
// older binaries that pre-date the field. Returns the number of records
// updated.
func backfillRuntimeTypes(store *state.Store, hermesImageName string, lookup imageLookupFunc) (int, error) {
	fixed := 0
	for _, inst := range store.Snapshot() {
		if inst.RuntimeType != "" {
			continue
		}
		imageRef, err := lookup(inst.ContainerID)
		if err != nil {
			log.Printf("backfill: skip %s: %v", inst.Name, err)
			continue
		}
		store.SetRuntimeType(inst.Name, runtimeFromImage(imageRef, hermesImageName))
		fixed++
	}
	if fixed == 0 {
		return 0, nil
	}
	return fixed, store.Save()
}

// runMigrations runs one-shot data migrations at server startup. Failures are
// logged but never block the server from starting.
func (s *Server) runMigrations() {
	store, err := s.loadStore()
	if err != nil {
		log.Printf("migrations: load store failed: %v", err)
		return
	}
	n, err := backfillRuntimeTypes(store, s.config.Hermes.ImageName, func(cid string) (string, error) {
		return container.ImageOf(s.docker, cid)
	})
	if err != nil {
		log.Printf("migrations: backfill runtime_type save failed: %v", err)
	}
	if n > 0 {
		log.Printf("migrations: backfilled runtime_type for %d instance(s)", n)
	}
}
