// Package all is a convenience wrapper that registers all known scale implementations.
// Importing this package enables the goscale factory to find drivers for any
// supported scale brand.
package all

// Import each implementation package for its side-effects (the init() function).
// The path now reflects the new 'pkg/scales/' structure.
import (
	_ "github.com/mlsorensen/goscale/pkg/scales/aku"
	_ "github.com/mlsorensen/goscale/pkg/scales/lunar"
	_ "github.com/mlsorensen/goscale/pkg/scales/mock"
	_ "github.com/mlsorensen/goscale/pkg/scales/themis"
	// When you add an [model] scale, you would add this line:
	// _ "github.com/mlsorensen/goscale/pkg/scales/[model]"
)
