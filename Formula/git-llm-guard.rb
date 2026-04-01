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

  def post_install
    config_path = Pathname.new(Dir.home)/".git-llm-guard.yaml"
    unless config_path.exist?
      system bin/"git-llm-guard", "--init"
      ohai "Default config created at #{config_path}"
      ohai "Edit the 'root' field to point to your shared directory before starting the service."
    end
  end

  service do
    run [opt_bin/"git-llm-guard", "--config", Pathname.new(Dir.home)/".git-llm-guard.yaml"]
    keep_alive true
    log_path var/"log/git-llm-guard.log"
    error_log_path var/"log/git-llm-guard.log"
    working_dir Dir.home
  end

  def caveats
    <<~EOS
      To use git-llm-guard as a service:

        1. Edit ~/.git-llm-guard.yaml and set 'root' to your shared directory
        2. Start the service:

           brew services start git-llm-guard

        3. Check logs at: #{var}/log/git-llm-guard.log
    EOS
  end

  test do
    assert_match "git-llm-guard", shell_output("#{bin}/git-llm-guard --help 2>&1", 0)
  end
end
