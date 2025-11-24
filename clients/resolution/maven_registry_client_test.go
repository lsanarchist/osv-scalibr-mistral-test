package resolution

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"deps.dev/util/maven"
	"deps.dev/util/resolve"
	"deps.dev/util/resolve/version"
	"github.com/google/go-cmp/cmp"
	"github.com/google/osv-scalibr/clients/datasource"
)

func TestMavenRegistryClient_Version(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/1.0.0/a-1.0.0.pom":
			fmt.Fprint(w, `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <version>1.0.0</version>
</project>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.Version(context.Background(), resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.Maven,
			Name:   "g:a",
		},
		Version:     "1.0.0",
		VersionType: resolve.Concrete,
	})
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}

	want := resolve.Version{
		VersionKey: resolve.VersionKey{
			PackageKey: resolve.PackageKey{
				System: resolve.Maven,
				Name:   "g:a",
			},
			Version:     "1.0.0",
			VersionType: resolve.Concrete,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Version() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_Versions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/maven-metadata.xml":
			fmt.Fprint(w, `<metadata>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <versioning>
    <versions>
      <version>1.0.0</version>
      <version>2.0.0</version>
    </versions>
  </versioning>
</metadata>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.Versions(context.Background(), resolve.PackageKey{
		System: resolve.Maven,
		Name:   "g:a",
	})
	if err != nil {
		t.Fatalf("Versions() error = %v", err)
	}

	want := []resolve.Version{
		{
			VersionKey: resolve.VersionKey{
				PackageKey: resolve.PackageKey{
					System: resolve.Maven,
					Name:   "g:a",
				},
				Version:     "1.0.0",
				VersionType: resolve.Concrete,
			},
		},
		{
			VersionKey: resolve.VersionKey{
				PackageKey: resolve.PackageKey{
					System: resolve.Maven,
					Name:   "g:a",
				},
				Version:     "2.0.0",
				VersionType: resolve.Concrete,
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Versions() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_Version_WithRepositories(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/1.0.0/a-1.0.0.pom":
			fmt.Fprint(w, `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <version>1.0.0</version>
  <repositories>
    <repository>
      <id>repo1</id>
      <url>https://repo1.example.com</url>
    </repository>
  </repositories>
</project>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.Version(context.Background(), resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.Maven,
			Name:   "g:a",
		},
		Version:     "1.0.0",
		VersionType: resolve.Concrete,
	})
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}

	var attr version.AttrSet
	attr.SetAttr(version.Registries, "dep:https://repo1.example.com")

	want := resolve.Version{
		VersionKey: resolve.VersionKey{
			PackageKey: resolve.PackageKey{
				System: resolve.Maven,
				Name:   "g:a",
			},
			Version:     "1.0.0",
			VersionType: resolve.Concrete,
		},
		AttrSet: attr,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Version() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_Requirements(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/1.0.0/a-1.0.0.pom":
			fmt.Fprint(w, `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <version>1.0.0</version>
  <dependencies>
    <dependency>
      <groupId>g2</groupId>
      <artifactId>a2</artifactId>
      <version>2.0.0</version>
    </dependency>
  </dependencies>
</project>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.Requirements(context.Background(), resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.Maven,
			Name:   "g:a",
		},
		Version:     "1.0.0",
		VersionType: resolve.Concrete,
	})
	if err != nil {
		t.Fatalf("Requirements() error = %v", err)
	}

	want := []resolve.RequirementVersion{
		{
			VersionKey: resolve.VersionKey{
				PackageKey: resolve.PackageKey{
					System: resolve.Maven,
					Name:   "g2:a2",
				},
				Version:     "2.0.0",
				VersionType: resolve.Requirement,
			},
			Type: resolve.MavenDepType(maven.Dependency{
				GroupID:    "g2",
				ArtifactID: "a2",
				Version:    "2.0.0",
			}, ""),
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Requirements() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_Requirements_WithParent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/1.0.0/a-1.0.0.pom":
			fmt.Fprint(w, `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <version>1.0.0</version>
  <parent>
    <groupId>g</groupId>
    <artifactId>p</artifactId>
    <version>1.0.0</version>
  </parent>
</project>`)
		case "/g/p/1.0.0/p-1.0.0.pom":
			fmt.Fprint(w, `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>g</groupId>
  <artifactId>p</artifactId>
  <version>1.0.0</version>
  <packaging>pom</packaging>
  <dependencies>
    <dependency>
      <groupId>g2</groupId>
      <artifactId>a2</artifactId>
      <version>2.0.0</version>
    </dependency>
  </dependencies>
</project>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.Requirements(context.Background(), resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.Maven,
			Name:   "g:a",
		},
		Version:     "1.0.0",
		VersionType: resolve.Concrete,
	})
	if err != nil {
		t.Fatalf("Requirements() error = %v", err)
	}

	want := []resolve.RequirementVersion{
		{
			VersionKey: resolve.VersionKey{
				PackageKey: resolve.PackageKey{
					System: resolve.Maven,
					Name:   "g2:a2",
				},
				Version:     "2.0.0",
				VersionType: resolve.Requirement,
			},
			Type: resolve.MavenDepType(maven.Dependency{
				GroupID:    "g2",
				ArtifactID: "a2",
				Version:    "2.0.0",
			}, ""),
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Requirements() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_MatchingVersions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/g/a/maven-metadata.xml":
			fmt.Fprint(w, `<metadata>
  <groupId>g</groupId>
  <artifactId>a</artifactId>
  <versioning>
    <versions>
      <version>1.0.0</version>
      <version>1.1.0</version>
      <version>2.0.0</version>
    </versions>
  </versioning>
</metadata>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c, err := NewMavenRegistryClient(context.Background(), ts.URL, "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	got, err := c.MatchingVersions(context.Background(), resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.Maven,
			Name:   "g:a",
		},
		Version:     "[1.0.0]",
		VersionType: resolve.Requirement,
	})
	if err != nil {
		t.Fatalf("MatchingVersions() error = %v", err)
	}

	want := []resolve.Version{
		{
			VersionKey: resolve.VersionKey{
				PackageKey: resolve.PackageKey{
					System: resolve.Maven,
					Name:   "g:a",
				},
				Version:     "1.0.0",
				VersionType: resolve.Concrete,
			},
		},
	}

	// Sort order might matter or not, usually MatchingVersions returns sorted or we should sort.
	// resolve.MatchRequirement usually returns sorted versions (descending or ascending?).
	// Let's check the output if it fails.
	// But wait, I should probably check if the order matches what I expect.
	// If I don't know the order, I can use cmp.Transformer to sort before comparing, or just check elements.
	// But let's assume some order.

	// Actually, let's just check if it contains the expected versions.
	// But cmp.Diff is strict about order for slices.

	// Let's run the test and see.
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MatchingVersions() mismatch (-want +got):\n%s", diff)
	}
}

func TestMavenRegistryClient_AddRegistries(t *testing.T) {
	c, err := NewMavenRegistryClient(context.Background(), "https://repo1.maven.org/maven2", "", true)
	if err != nil {
		t.Fatalf("NewMavenRegistryClient() error = %v", err)
	}

	err = c.AddRegistries(context.Background(), []Registry{
		datasource.MavenRegistry{
			URL: "https://repo2.maven.org/maven2",
			ID:  "repo2",
		},
	})
	if err != nil {
		t.Fatalf("AddRegistries() error = %v", err)
	}
}

func TestMavenRegistryClient_Errors(t *testing.T) {
	c, _ := NewMavenRegistryClient(context.Background(), "http://localhost", "", true)
	ctx := context.Background()

	// Version: invalid name
	_, err := c.Version(ctx, resolve.VersionKey{PackageKey: resolve.PackageKey{Name: "invalid"}})
	if err == nil {
		t.Error("Version() expected error for invalid name")
	}

	// Versions: wrong system
	_, err = c.Versions(ctx, resolve.PackageKey{System: resolve.NPM, Name: "g:a"})
	if err == nil {
		t.Error("Versions() expected error for wrong system")
	}

	// Versions: invalid name
	_, err = c.Versions(ctx, resolve.PackageKey{System: resolve.Maven, Name: "invalid"})
	if err == nil {
		t.Error("Versions() expected error for invalid name")
	}

	// Requirements: wrong system
	_, err = c.Requirements(ctx, resolve.VersionKey{PackageKey: resolve.PackageKey{System: resolve.NPM, Name: "g:a"}})
	if err == nil {
		t.Error("Requirements() expected error for wrong system")
	}

	// Requirements: invalid name
	_, err = c.Requirements(ctx, resolve.VersionKey{PackageKey: resolve.PackageKey{System: resolve.Maven, Name: "invalid"}})
	if err == nil {
		t.Error("Requirements() expected error for invalid name")
	}

	// MatchingVersions: wrong system
	_, err = c.MatchingVersions(ctx, resolve.VersionKey{PackageKey: resolve.PackageKey{System: resolve.NPM, Name: "g:a"}})
	if err == nil {
		t.Error("MatchingVersions() expected error for wrong system")
	}

	// AddRegistries: invalid type
	err = c.AddRegistries(ctx, []Registry{"invalid"})
	if err == nil {
		t.Error("AddRegistries() expected error for invalid type")
	}
}

func TestNewMavenRegistryClientWithAPI(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewMavenRegistryClientWithAPI(nil) did not panic")
		}
	}()
	NewMavenRegistryClientWithAPI(nil)
}

func TestNewMavenRegistryClientWithAPI_Success(t *testing.T) {
	api, _ := datasource.NewMavenRegistryAPIClient(context.Background(), datasource.MavenRegistry{URL: "http://localhost"}, "", true)
	c := NewMavenRegistryClientWithAPI(api)
	if c == nil {
		t.Error("NewMavenRegistryClientWithAPI() returned nil")
	}
}
