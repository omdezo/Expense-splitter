package types

import "go.uber.org/zap"

// Logger is the application logger, aliased so the rest of the codebase depends
// on types.Logger rather than zap directly. Used as *types.Logger.
type Logger = zap.SugaredLogger
