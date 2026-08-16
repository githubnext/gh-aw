package generatedyamlheredoc

func generatedWorkflowFragments() []string {
	return []string{
		"          cat > config.json << 'EOF'\n{}\nEOF\n", // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"cat <<EOF | node renderer.cjs\n{}\nEOF\n",        // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"cat <<-EOF\ncontent\nEOF\n",                      // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"tee config.json <<'EOF'\n{}\nEOF\n",              // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"node renderer.cjs <<EOF\n{}\nEOF\n",              // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"read VALUE << EOF\nvalue\nEOF\n",                 // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"<<EOF command\ncontent\nEOF\n",                   // want `generated workflow shell uses a heredoc; pass plain content through an environment variable to a JavaScript renderer instead \(do not base64 encode it\)`
		"cat <<< \"$VALUE\"\n",
		"echo $((1 << 2))\n",
		`(?ms)<<\s*EOF`,
		"echo no-heredoc\n",
	}
}

func suppressedHeredoc() string {
	//nolint:generatedyamlheredoc // Existing migration debt is tracked explicitly.
	return "cat > file <<'EOF'\ncontent\nEOF\n"
}
