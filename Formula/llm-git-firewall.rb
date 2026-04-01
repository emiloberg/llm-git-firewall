class LlmGitFirewall < Formula
  desc "Gatekeeper daemon between AI coding agents and git"
  homepage "https://github.com/emiloberg/llm-git-firewall"
  url "https://github.com/emiloberg/llm-git-firewall/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "3440951da9c912a0bde9c90add49637d5787e9c8bcbe1de2770e2e0eca0440c3"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/llm-git-firewall"
  end

  def post_install
    config_path = Pathname.new(Dir.home)/".llm-git-firewall.yaml"
    unless config_path.exist?
      system bin/"llm-git-firewall", "--init"
      ohai "Default config created at #{config_path}"
      ohai "Edit the 'root' field to point to your shared directory before starting the service."
    end
  end

  service do
    run [opt_bin/"llm-git-firewall", "--config", Pathname.new(Dir.home)/".llm-git-firewall.yaml"]
    keep_alive true
    log_path var/"log/llm-git-firewall.log"
    error_log_path var/"log/llm-git-firewall.log"
    working_dir Dir.home
  end

  def caveats
    <<~EOS
      To use llm-git-firewall as a service:

        1. Edit ~/.llm-git-firewall.yaml and set 'root' to your shared directory
        2. Start the service:

           brew services start llm-git-firewall

        3. Check logs at: #{var}/log/llm-git-firewall.log
    EOS
  end

  test do
    assert_match "llm-git-firewall", shell_output("#{bin}/llm-git-firewall --help 2>&1", 0)
  end
end
