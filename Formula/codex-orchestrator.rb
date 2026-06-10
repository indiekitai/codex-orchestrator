class CodexOrchestrator < Formula
  desc "Codex App-first orchestration helper for supervised multi-session development"
  homepage "https://github.com/indiekitai/codex-orchestrator"
  version "0.3.0-beta.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/indiekitai/codex-orchestrator/releases/download/v0.3.0-beta.1/codex-orchestrator_darwin_arm64.tar.gz"
      sha256 "bc217cc2ee0e3f9e969ee599607156bfb76cf7a8eb9aa237a90ccd87fc0f6556"
    end

    on_intel do
      url "https://github.com/indiekitai/codex-orchestrator/releases/download/v0.3.0-beta.1/codex-orchestrator_darwin_amd64.tar.gz"
      sha256 "87a594826d11b0cd30c21fc8f3dd9c5f5f449db15ca4cfae86a2f82f0e7c3f0b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/indiekitai/codex-orchestrator/releases/download/v0.3.0-beta.1/codex-orchestrator_linux_arm64.tar.gz"
      sha256 "860951b645a673240d0abd579f57648dde4e5dd20fe995e94a4ae1cf9f54d253"
    end

    on_intel do
      url "https://github.com/indiekitai/codex-orchestrator/releases/download/v0.3.0-beta.1/codex-orchestrator_linux_amd64.tar.gz"
      sha256 "dbf0a45770111e798b959a3df4e5a57164229d33811ff2777a5490f9dce53c52"
    end
  end

  def install
    bin.install Dir["codex-orchestrator_*"].first => "codex-orchestrator"
    generate_completions_from_executable(bin/"codex-orchestrator", "completion")
  end

  test do
    assert_match "codex-orchestrator", shell_output("#{bin}/codex-orchestrator --help")
    assert_match "release-verifier", shell_output("#{bin}/codex-orchestrator completion bash")
  end
end
