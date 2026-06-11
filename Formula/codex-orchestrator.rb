class CodexOrchestrator < Formula
  desc "Codex App-first orchestration helper for supervised multi-session development"
  homepage "https://github.com/indiekitai/codex-orchestrator"
  url "https://github.com/indiekitai/codex-orchestrator.git",
      tag: "v0.3.0-beta.3"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-ldflags=-s -w", "-o", bin/"codex-orchestrator", "./cmd/codex-orchestrator"
    generate_completions_from_executable(bin/"codex-orchestrator", "completion")
  end

  test do
    assert_match "codex-orchestrator", shell_output("#{bin}/codex-orchestrator --help")
    assert_match "release-verifier", shell_output("#{bin}/codex-orchestrator completion bash")
  end
end
