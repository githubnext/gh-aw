package workflow

import "testing"

func BenchmarkValidateTemplateInjectionFastPath(b *testing.B) {
	compiler := NewCompiler()

	yamlContent := `
name: generated-safe-expression
on: workflow_dispatch
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Allowed generated expression
        run: |
          node ${{ runner.temp }}/actions/foo.cjs
          echo "${{ env.GH_AW_INPUT }}"
          echo done
`

	b.ReportAllocs()
	for range b.N {
		if err := compiler.validateTemplateInjection(yamlContent, "", "", nil); err != nil {
			b.Fatalf("validateTemplateInjection() error = %v", err)
		}
	}
}
