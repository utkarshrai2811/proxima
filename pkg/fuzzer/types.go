// Package fuzzer implements an Intruder-style request fuzzer: it substitutes
// payloads into §marker§ positions of a base request and sends the generated
// requests concurrently, streaming results.
package fuzzer

import "time"

type (
	AttackType        string
	AttackStatus      string
	PayloadSourceType string
	BuiltInList       string
)

const (
	AttackTypeSniper       AttackType = "SNIPER"
	AttackTypeBatteringRam AttackType = "BATTERING_RAM"
	AttackTypePitchfork    AttackType = "PITCHFORK"
	AttackTypeClusterBomb  AttackType = "CLUSTER_BOMB"

	AttackStatusPending   AttackStatus = "PENDING"
	AttackStatusRunning   AttackStatus = "RUNNING"
	AttackStatusPaused    AttackStatus = "PAUSED"
	AttackStatusDone      AttackStatus = "DONE"
	AttackStatusCancelled AttackStatus = "CANCELLED"

	PayloadSourceInline  PayloadSourceType = "INLINE"
	PayloadSourceBuiltIn PayloadSourceType = "BUILT_IN"

	BuiltInSQLiBasic       BuiltInList = "SQLI_BASIC"
	BuiltInXSSBasic        BuiltInList = "XSS_BASIC"
	BuiltInCommonPasswords BuiltInList = "COMMON_PASSWORDS"
	BuiltInDirNames        BuiltInList = "DIR_NAMES"
	BuiltInNumericRange    BuiltInList = "NUMERIC_RANGE"
)

// PayloadSource describes where the payloads for a position come from.
type PayloadSource struct {
	Type     PayloadSourceType
	Values   []string    // INLINE
	BuiltIn  BuiltInList // BUILT_IN
	RangeMin int
	RangeMax int
}

// Resolve returns the concrete payload list for this source.
func (s PayloadSource) Resolve() []string {
	if s.Type == PayloadSourceBuiltIn {
		return BuiltInPayloads(s.BuiltIn, s.RangeMin, s.RangeMax)
	}

	return s.Values
}

type Attack struct {
	ID             string
	Name           string
	Type           AttackType
	BaseRequest    string // raw HTTP with §markers§
	PayloadSources []PayloadSource
	Concurrency    int
	Status         AttackStatus
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	TotalRequests  int
	CompletedCount int
	ErrorCount     int
}

type FuzzResult struct {
	ID             string
	AttackID       string
	RequestIndex   int
	PayloadValues  map[string]string // positionName → value
	RawRequest     string
	RawResponse    string
	StatusCode     int
	ResponseSize   int
	ResponseTimeMs int64
	IsError        bool
	ErrorMessage   string
}
