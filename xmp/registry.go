// Copyright 2026 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xmp

import "fmt"

var _registries = make(map[string]InjectFunc)

// InjectFunc injects xmpData into imageData and returns the updated image bytes.
type InjectFunc = func(imageData, xmpData []byte) ([]byte, error)

// RegisterInjectFunc registers an XMP injector for the image type.
// It panics if imageType is empty or inject is nil.
func RegisterInjectFunc(imageType string, inject InjectFunc) {
	if imageType == "" {
		panic("xmp.RegisterInjectFunc: imageType is empty")
	}
	if inject == nil {
		panic("xmp.RegisterInjectFunc: inject is nil")
	}
	_registries[imageType] = inject
}

// Inject injects xmpData into imageData using the registered injector.
func Inject(imageType string, imageData, xmpData []byte) ([]byte, error) {
	if inject, ok := _registries[imageType]; ok {
		return inject(imageData, xmpData)
	}
	return nil, fmt.Errorf("not found the xmp injector for image type '%s'", imageType)
}
