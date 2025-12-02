package sqlite3pkg

import (
	"fmt"
	"strings"
	"time"
)

// Mode Durability  Performance  Journal Notes
const (
	DeleteJournalMode   JournalMode = "DELETE"   // Safe  	Slowest   Creates/deletes Default
	TruncateJournalMode JournalMode = "TRUNCATE" // Safe  	Moderate  Truncate file
	PersistJournalMode  JournalMode = "PERSIST"  // Safe  	Moderate  Reuses file			Old data lingers
	MemoryJournalMode   JournalMode = "MEMORY"   // Unsafe  Fast      In RAM only			Crash can corrupt DB
	WALJournalMode      JournalMode = "WAL"      // Safe  	HighPerf  WAL file			  Many readers, Best concurrency
	NoJournalMode       JournalMode = "OFF"      // Unsafe  Fastest   No journal			Corruption risk
)

type JournalMode string

func ParseJournalMode(value string) (jm JournalMode, err error) {
	jm = JournalMode(strings.ToUpper(value))
	switch jm {
	case DeleteJournalMode, MemoryJournalMode, PersistJournalMode, TruncateJournalMode, WALJournalMode, NoJournalMode:
		// S'all good
	case "":
		jm = DefaultJournalMode
	default:
		err = fmt.Errorf("invalid journal mode: %s", jm)
		jm = ""
	}
	return jm, err
}

const (
	NoSynchronous     Synchronous = "OFF"    // Never fsync(). Fastest. A crash/power-loss can corrupt the DB.
	NormalSynchronous Synchronous = "NORMAL" // Fsync infrequently. DB consistent after a crash, may lose last transactions.
	FullSynchronous   Synchronous = "FULL"   // Fsync frequently. After COMMIT, transaction should be durable.
	ExtraSynchronous  Synchronous = "EXTRA"  // Like FULL plus extra syncs (e.g., dir and journal headers).Slower/safer than FULL.
)

type Synchronous string

func ParseSynchronous(value string) (s Synchronous, err error) {
	s = Synchronous(strings.ToUpper(value))
	switch s {
	case NoSynchronous, NormalSynchronous, FullSynchronous, ExtraSynchronous:
		// S'all good
	case "":
		s = DefaultSynchronous
	default:
		err = fmt.Errorf("invalid synchronous value: %s", s)
		s = ""
	}
	return s, err
}

const (
	EnforceForeignKeys ForeignKeyMode = "ON"
	IgnoreForeignKeys  ForeignKeyMode = "OFF"
)

type ForeignKeyMode string

func ParseForeignKeyMode(value string) (fkm ForeignKeyMode, err error) {
	fkm = ForeignKeyMode(strings.ToUpper(value))
	switch fkm {
	case EnforceForeignKeys, IgnoreForeignKeys:
		// S'all good
	case "":
		fkm = DefaultForeignKeyMode
	default:
		err = fmt.Errorf("invalid foreign key mode: %s", fkm)
		fkm = ""
	}
	return fkm, err
}

func ParseAutoCheckpoint(value int) (cp int, err error) {
	if value == 0 {
		cp = DefaultWALAutoCheckpoint
		goto end
	}
	if value < 0 {
		err = NewErr(ErrInvalidWALAutocheckpointValue, "wal_autocheckpoint", value, err)
		goto end
	}
	cp = value
end:
	return cp, err
}

func ParseBusyTimeout(value int) (bt time.Duration, err error) {
	if value == 0 {
		bt = DefaultBusyTimeout
		goto end
	}
	if value < 0 {
		err = NewErr(ErrInvalidBusyTimeoutValue, "busy_timeout", value, err)
		goto end
	}
	bt = time.Duration(value) * time.Second / time.Millisecond
end:
	return bt, err
}
