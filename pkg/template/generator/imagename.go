package generator

import (
	imgtypes "github.com/openshift/origin/pkg/image/api"
)

// ImageNameGenerator implements Generator interface. It generates
// an image name based on the input component and the master's default image format.
//
// Examples:
//
// from             | value
// -----------------------------
// "router"         | "registry.access.redhat.com/openshift3/ose-router:latest"
//
type ImageNameGenerator struct {
}

// NewImageNameGenerator creates new ImageNameGenerator.
func NewImageNameGenerator() ImageNameGenerator {
	return ImageNameGenerator{}
}

// GenerateValue generates the name based on the input.
func (g ImageNameGenerator) GenerateValue(component string) (interface{}, error) {
	return imgtypes.ImageFor(component), nil
}
