package modulegen

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var packageNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

var writeExistingFile = writeExistingFileAtomically

type Options struct {
	Root   string
	Domain string
	Module string
}

type Result struct {
	Created []string
	Updated []string
}

func normalizeOptions(opts Options) (Options, error) {
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.Domain == "" {
		opts.Domain = "admin"
	}
	if opts.Module == "" {
		opts.Module = "demo"
	}
	if !validPackageName(opts.Domain) {
		return Options{}, fmt.Errorf("invalid domain %q", opts.Domain)
	}
	if opts.Domain == "middleware" {
		return Options{}, fmt.Errorf("reserved domain %q", opts.Domain)
	}
	if opts.Domain == "main" {
		return Options{}, fmt.Errorf("reserved domain %q", opts.Domain)
	}
	if !validPackageName(opts.Module) {
		return Options{}, fmt.Errorf("invalid module %q", opts.Module)
	}
	if opts.Module == "main" {
		return Options{}, fmt.Errorf("reserved module %q", opts.Module)
	}
	return opts, nil
}

func validPackageName(name string) bool {
	return packageNamePattern.MatchString(name) && !token.Lookup(name).IsKeyword()
}

// Generate creates and registers a business module skeleton.
func Generate(input Options) (*Result, error) {
	opts, err := normalizeOptions(input)
	if err != nil {
		return nil, err
	}
	modulePath, err := readModulePath(opts.Root)
	if err != nil {
		return nil, err
	}
	typeName := exportedName(opts.Module)
	containerName := typeName + "Container"
	if opts.Domain != "admin" {
		containerName = exportedName(opts.Domain) + containerName
	}
	data := templateData{
		ModulePath:    modulePath,
		Domain:        opts.Domain,
		Module:        opts.Module,
		TypeName:      typeName,
		ContainerName: containerName,
		DaoAlias:      opts.Domain + exportedName(opts.Module) + "Dao",
		ServiceAlias:  opts.Domain + exportedName(opts.Module) + "Service",
		RouterAlias:   opts.Domain + "Router",
	}

	paths := map[string]string{
		"dao":     filepath.Join("internal", "dao", opts.Domain, opts.Module, opts.Module+".go"),
		"service": filepath.Join("internal", "service", opts.Domain, opts.Module, opts.Module+".go"),
		"api":     filepath.Join("internal", "api", opts.Domain, opts.Module, opts.Module+".go"),
		"router":  filepath.Join("internal", "router", opts.Domain, opts.Module+"_router.go"),
	}

	rendered := make(map[string][]byte, len(paths))
	for _, name := range sortedStringMapKeys(paths) {
		relative := paths[name]
		path := filepath.Join(opts.Root, relative)
		if _, err := os.Stat(path); err == nil {
			return nil, fmt.Errorf("module already exists: %s", relative)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check %s: %w", relative, err)
		}
		content, err := renderModuleTemplate(name, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", relative, err)
		}
		rendered[relative] = content
	}

	updated := make(map[string][]byte, 3)
	originals := make(map[string][]byte, 3)
	diRelative := filepath.Join("internal", "di", "container.go")
	diSource, err := os.ReadFile(filepath.Join(opts.Root, diRelative))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", diRelative, err)
	}
	updated[diRelative], err = updateDI(diSource, data)
	if err != nil {
		return nil, err
	}
	originals[diRelative] = diSource

	domainRouterRelative := filepath.Join("internal", "router", opts.Domain, "router.go")
	domainRouterPath := filepath.Join(opts.Root, domainRouterRelative)
	domainRouterSource, err := os.ReadFile(domainRouterPath)
	if err == nil {
		updated[domainRouterRelative], err = updateDomainRouter(domainRouterSource, data)
		if err != nil {
			return nil, err
		}
		originals[domainRouterRelative] = domainRouterSource
	} else if os.IsNotExist(err) {
		rendered[domainRouterRelative], err = renderDomainRouter(data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", domainRouterRelative, err)
		}
	} else {
		return nil, fmt.Errorf("read %s: %w", domainRouterRelative, err)
	}

	rootRouterRelative := filepath.Join("internal", "router", "router.go")
	rootRouterSource, err := os.ReadFile(filepath.Join(opts.Root, rootRouterRelative))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rootRouterRelative, err)
	}
	rootRouterContent, rootRouterChanged, err := updateRootRouter(rootRouterSource, data)
	if err != nil {
		return nil, err
	}
	if rootRouterChanged {
		updated[rootRouterRelative] = rootRouterContent
		originals[rootRouterRelative] = rootRouterSource
	}

	created := make([]string, 0, len(rendered))
	for _, relative := range sortedByteMapKeys(rendered) {
		content := rendered[relative]
		path := filepath.Join(opts.Root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			cleanupCreatedFiles(opts.Root, created)
			return nil, fmt.Errorf("create directory for %s: %w", relative, err)
		}
		if err := writeNewFile(path, content); err != nil {
			cleanupCreatedFiles(opts.Root, created)
			if os.IsExist(err) {
				return nil, fmt.Errorf("module already exists: %s", relative)
			}
			return nil, fmt.Errorf("write %s: %w", relative, err)
		}
		created = append(created, relative)
	}
	updatedPaths := make([]string, 0, len(updated))
	for _, relative := range sortedByteMapKeys(updated) {
		content := updated[relative]
		if err := writeExistingFile(filepath.Join(opts.Root, relative), content); err != nil {
			cleanupCreatedFiles(opts.Root, created)
			if rollbackErr := rollbackUpdatedFiles(opts.Root, updatedPaths, originals); rollbackErr != nil {
				return nil, fmt.Errorf("write %s: %v; rollback updated files: %w", relative, err, rollbackErr)
			}
			return nil, fmt.Errorf("write %s: %w", relative, err)
		}
		updatedPaths = append(updatedPaths, relative)
	}
	return &Result{Created: created, Updated: updatedPaths}, nil
}

func sortedStringMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedByteMapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeNewFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func writeExistingFileAtomically(path string, content []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(filepath.Dir(path), ".modulegen-*")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	if err := file.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	closed = true
	return os.Rename(temporaryPath, path)
}

func cleanupCreatedFiles(root string, relativePaths []string) {
	for _, relative := range relativePaths {
		_ = os.Remove(filepath.Join(root, relative))
	}
}

func rollbackUpdatedFiles(root string, relativePaths []string, originals map[string][]byte) error {
	var restoreErrors []error
	for index := len(relativePaths) - 1; index >= 0; index-- {
		relative := relativePaths[index]
		if err := writeExistingFileAtomically(filepath.Join(root, relative), originals[relative]); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf("restore %s: %w", relative, err))
		}
	}
	return errors.Join(restoreErrors...)
}

func readModulePath(root string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "module ") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if path != "" {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("module path not found in go.mod")
}

func exportedName(name string) string {
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
