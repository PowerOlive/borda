package borda

import (
	"time"

	"github.com/getlantern/zenodb"
)

// TDBSave creates a SaveFN that saves to an embedded tdb.DB
func TDBSave(dir string, schemaFile string) (SaveFunc, *zenodb.DB, error) {
	db, err := zenodb.NewDB(&zenodb.DBOpts{
		Dir:        dir,
		SchemaFile: schemaFile,
	})
	if err != nil {
		return nil, nil, err
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			for name := range db.AllTableStats() {
				db.PrintTableStats(name)
			}
		}
	}()

	return func(m *Measurement) error {
		return db.Insert("inbound", &zenodb.Point{
			Ts:   time.Now(), // use now so that clients with bad clocks don't throw things off m.Ts,
			Dims: m.Dimensions,
			Vals: m.Values,
		})
	}, db, nil
}
