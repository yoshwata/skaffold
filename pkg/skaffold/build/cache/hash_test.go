/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type stubDependencyLister struct {
	dependencies []string
}

func (m *stubDependencyLister) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return m.dependencies, nil
}

var mockCacheHasher = func(s string) (string, error) {
	return s, nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		description  string
		dependencies [][]string
		expected     string
	}{
		{
			description: "check dependencies in different orders",
			dependencies: [][]string{
				{"a", "b"},
				{"b", "a"},
			},
			expected: "eb394fd4559b1d9c383f4359667a508a615b82a74e1b160fce539f86ae0842e8",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&hashFunction, mockCacheHasher)

			for _, d := range test.dependencies {
				depLister := &stubDependencyLister{dependencies: d}
				actual, err := getHashForArtifact(context.Background(), depLister, nil)

				t.CheckNoError(err)
				t.CheckDeepEqual(test.expected, actual)
			}
		})
	}
}

func TestCacheHasher(t *testing.T) {
	tests := []struct {
		description   string
		differentHash bool
		newFilename   string
		update        func(oldFile string, folder *testutil.TempDir)
	}{
		{
			description:   "change filename",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
			},
		},
		{
			description:   "change file contents",
			differentHash: true,
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			description:   "change both",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			description:   "change nothing",
			differentHash: false,
			update:        func(oldFile string, folder *testutil.TempDir) {},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			originalFile := "foo"
			originalContents := "contents"

			tmpDir := t.NewTempDir().
				Write(originalFile, originalContents)

			path := originalFile
			depLister := &stubDependencyLister{dependencies: []string{tmpDir.Path(originalFile)}}

			oldHash, err := getHashForArtifact(context.Background(), depLister, nil)
			t.CheckNoError(err)

			test.update(originalFile, tmpDir)
			if test.newFilename != "" {
				path = test.newFilename
			}

			depLister = &stubDependencyLister{dependencies: []string{tmpDir.Path(path)}}
			newHash, err := getHashForArtifact(context.Background(), depLister, nil)

			t.CheckNoError(err)
			t.CheckDeepEqual(false, test.differentHash && oldHash == newHash)
			t.CheckDeepEqual(false, !test.differentHash && oldHash != newHash)
		})
	}
}
