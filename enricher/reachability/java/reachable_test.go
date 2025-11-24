package java

import (
	"testing"
)

func TestIsDynamicCodeLoading(t *testing.T) {
	tests := []struct {
		method     string
		descriptor string
		want       bool
	}{
		{"loadClass", "(Ljava/lang/String;)Ljava/lang/Class;", true},
		{"forName", "(Ljava/lang/String;)Ljava/lang/Class;", true},
		{"loadClass", "(Ljava/lang/String;)V", false},
		{"otherMethod", "(Ljava/lang/String;)Ljava/lang/Class;", false},
		{"forName", "(Ljava/lang/String;)V", false},
	}

	for _, tt := range tests {
		if got := isDynamicCodeLoading(tt.method, tt.descriptor); got != tt.want {
			t.Errorf("isDynamicCodeLoading(%q, %q) = %v, want %v", tt.method, tt.descriptor, got, tt.want)
		}
	}
}

func TestIsDependencyInjection(t *testing.T) {
	tests := []struct {
		class string
		want  bool
	}{
		{"javax/inject/Inject", true},
		{"org/springframework/beans/factory/annotation/Autowired", true},
		{"com/google/inject/Inject", true},
		{"dagger/Component", true},
		{"java/lang/String", false},
		{"com/example/MyClass", false},
	}

	for _, tt := range tests {
		if got := isDependencyInjection(tt.class); got != tt.want {
			t.Errorf("isDependencyInjection(%q) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestNewReachabilityEnumerator(t *testing.T) {
	classPaths := []string{"path/to/classes"}
	enumerator := NewReachabilityEnumerator(classPaths, nil, DontHandleDynamicCode, DontHandleDynamicCode)

	if len(enumerator.ClassPaths) != 1 || enumerator.ClassPaths[0] != "path/to/classes" {
		t.Errorf("ClassPaths = %v, want %v", enumerator.ClassPaths, classPaths)
	}
	if enumerator.CodeLoadingStrategy != DontHandleDynamicCode {
		t.Errorf("CodeLoadingStrategy = %v, want %v", enumerator.CodeLoadingStrategy, DontHandleDynamicCode)
	}
	if enumerator.DependencyInjectionStrategy != DontHandleDynamicCode {
		t.Errorf("DependencyInjectionStrategy = %v, want %v", enumerator.DependencyInjectionStrategy, DontHandleDynamicCode)
	}
	if enumerator.loadedJARs == nil {
		t.Error("loadedJARs is nil")
	}
}
