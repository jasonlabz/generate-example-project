	"sort"
	var sourceRoots pathList
	flag.Var(&sourceRoots, "path", "source directory tree to scan")
	for index, path := range sourceRoots {
		sourceRoot, err := filepath.Abs(path)
		if err != nil {
			log.Fatalf("resolve source path %q: %v", path, err)
		}
		if !isWithin(sourceRoot, rootPath) {
			log.Fatalf("source path %q must be within project root %q", sourceRoot, rootPath)
		}
		sourceRoots[index] = sourceRoot
	}
	if len(sourceRoots) == 0 {
		sourceRoots = append(sourceRoots, rootPath)
	}

