class GitLlmGuard < Formula
  desc "Gatekeeper daemon between AI coding agents and git"
  homepage "https://github.com/git-llm-guard/git-llm-guard"
  url "https://github.com/git-llm-guard/git-llm-guard/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/git-llm-guard"
  end

  service do
    run [opt_bin/"git-llm-guard"]
    keep_alive true
    log_path var/"log/git-llm-guard.log"
    error_log_path var/"log/git-llm-guard.log"
  end

  test do
    assert_match "git-llm-guard", shell_output("#{bin}/git-llm-guard --help 2>&1", 0)
  end
end
