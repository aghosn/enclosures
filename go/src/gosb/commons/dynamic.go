package commons

/* This is specific to the dynamic usage of LitterBox */

var (
	PythonRuntime = map[string]bool{
		"builtins":            true,
		"sys":                 true,
		"importlib.abc":       true,
		"mpl_toolkits":        true,
		"mhdefault":           true,
		"importlib.util":      true,
		"site":                true,
		"zipimport":           true,
		"importlib.machinery": true,
	}

	PythonFix = map[string]bool{
		"__main__": true,
	}

	/* These are synthetic packages that do not actually exist.*/
	PythonSynthetic = map[string]bool{
		"_bootstrap":          true,
		"_bootstrap_external": true,
		"_io":                 true,
		"sitecustomize":       true,
		"usercustomize":       true,
		"":                    true,
	}
)
