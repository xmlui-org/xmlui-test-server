package minion

import (
	"errors"
	"fmt"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtglob"
	"github.com/xmlui-org/xmlui-test-server/xmluisvr/svrcfg"
)

// ParseManifest converts raw svrcfg.Manifest to type-checked runpkg.Manifest
func ParseManifest(raw *cfgldr.Manifest) (manifest *Manifest, err error) {
	var schema dt.URL
	var slug dt.URLSegment
	var branch dt.Identifier
	var tag dt.Identifier
	var subdir dt.PathSegments
	var repo dt.URLSegments
	var copyRules []CopyRule
	var variants []Variant
	var sourceType SourceType

	// Parse Schema URL
	schema, err = dt.ParseURL(raw.Schema)
	if err != nil {
		err = fmt.Errorf("invalid schema URL: %w", err)
		goto end
	}

	// Parse Slug (URL-safe segment)
	slug, err = dt.ParseURLSegment(raw.Slug)
	if err != nil {
		err = fmt.Errorf("invalid slug: %w", err)
		goto end
	}

	// Parse Subdir if present
	if raw.Source.Subdir != "" {
		subdir = dt.PathSegments(raw.Source.Subdir)
	}

	// Convert copy copyRules
	copyRules, err = parseCopyRules(raw.Copy)
	if err != nil {
		goto end
	}

	// Convert variants
	variants, err = parseVariants(raw.Variants)
	if err != nil {
		goto end
	}

	// Parse source type
	sourceType, err = ParseSourceType(raw.Source.Type)
	if err != nil {
		err = fmt.Errorf("invalid source type: %w", err)
		goto end
	}

	repo, err = dt.ParseURLSegments(raw.Source.Repo)
	if err != nil {
		err = fmt.Errorf("invalid repo: %w", err)
		goto end
	}

	branch, err = dt.ParseIdentifier(raw.Source.Branch)
	if err != nil && !errors.Is(err, dt.ErrEmpty) {
		err = fmt.Errorf("invalid Git branch: %w", err)
		goto end
	}

	tag, err = dt.ParseIdentifier(raw.Source.Branch)
	if err != nil && !errors.Is(err, dt.ErrEmpty) {
		err = fmt.Errorf("invalid Git tag: %w", err)
		goto end
	}

	if branch != "" && tag != "" {
		err = fmt.Errorf("cannot have both a branch ('%s') and a tag ('%s')", branch, tag)
		goto end
	}

	manifest = &Manifest{
		Schema:      schema,
		Version:     raw.Version,
		Slug:        slug,
		Name:        raw.Name,
		Description: raw.Description,
		Source: DemoSource{
			Type:   sourceType,
			Repo:   repo,
			Ref:    branch + tag,
			Subdir: subdir,
		},
		Copy:     copyRules,
		Variants: variants,
	}

	// Validate manifest
	err = validateManifest(manifest)
	if err != nil {
		goto end
	}

end:
	return manifest, err
}

// parseCopyRules converts raw copy rules to type-checked copy rules
func parseCopyRules(rawRules []cfgldr.CopyRule) (rules []CopyRule, err error) {
	var rawRule cfgldr.CopyRule
	var i int

	rules = make([]CopyRule, len(rawRules))
	for i, rawRule = range rawRules {
		rules[i] = CopyRule{
			From:     rawRule.From,
			To:       rawRule.To,
			Optional: rawRule.Optional,
		}
	}

	return rules, err
}

// parseVariants converts raw variants to type-checked variants
func parseVariants(rawVariants []cfgldr.Variant) (variants []Variant, err error) {
	var rawVariant cfgldr.Variant
	var slug dt.URLSegment
	var copyRules []CopyRule
	var i int

	variants = make([]Variant, len(rawVariants))
	for i, rawVariant = range rawVariants {
		slug, err = dt.ParseURLSegment(rawVariant.Slug)
		if err != nil {
			err = fmt.Errorf("invalid variant slug: %w", err)
			goto end
		}

		copyRules, err = parseCopyRules(rawVariant.Copy)
		if err != nil {
			goto end
		}

		variants[i] = Variant{
			Slug: slug,
			Name: rawVariant.Name,
			Copy: copyRules,
		}
	}

end:
	return variants, err
}

// validateManifest performs basic validation on the manifest structure
func validateManifest(m *Manifest) (err error) {
	var validTypes map[SourceType]bool

	if m.Slug == "" {
		err = fmt.Errorf("manifest missing required field: slug")
		goto end
	}
	if m.Name == "" {
		err = fmt.Errorf("manifest missing required field: name")
		goto end
	}
	if m.Source.Type == "" {
		err = fmt.Errorf("manifest missing required field: source.type")
		goto end
	}
	if m.Source.Repo == "" {
		err = fmt.Errorf("manifest missing required field: source.repo")
		goto end
	}

	// Validate source type
	validTypes = map[SourceType]bool{
		GitHubSourceType: true,
		URLSourceType:    true,
	}
	if !validTypes[m.Source.Type] {
		err = fmt.Errorf("unsupported source type '%s' (supported: github, url)", m.Source.Type)
		goto end
	}

	// At least one copy rule is recommended
	if len(m.Copy) == 0 {
		err = fmt.Errorf("manifest has no copy rules")
		goto end
	}

end:
	return err
}

// ParseGlobRules converts application CopyRule to dtglob.GlobRules
func ParseGlobRules(rules []CopyRule, baseDir dt.DirPath) (globRules *dtglob.GlobRules, err error) {
	var dtRules []dtglob.GlobRule
	var i int
	var r CopyRule

	dtRules = make([]dtglob.GlobRule, len(rules))
	for i, r = range rules {
		dtRules[i] = dtglob.GlobRule{
			From:     dtglob.Glob(r.From),
			To:       dt.EntryPath(r.To),
			Optional: r.Optional,
		}
	}

	globRules = &dtglob.GlobRules{
		BaseDir: baseDir,
		Rules:   dtRules,
	}

	return globRules, err
}
