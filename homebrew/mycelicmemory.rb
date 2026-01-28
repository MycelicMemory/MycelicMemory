class Mycelicmemory < Formula
  desc "AI-powered persistent memory system for Claude and other AI agents"
  homepage "https://github.com/MycelicMemory/mycelicmemory"
  version "1.2.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/MycelicMemory/mycelicmemory/releases/download/v1.2.2/mycelicmemory-macos-arm64"
      sha256 "PLACEHOLDER_ARM64_SHA256"
    else
      url "https://github.com/MycelicMemory/mycelicmemory/releases/download/v1.2.2/mycelicmemory-macos-x64"
      sha256 "PLACEHOLDER_X64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/MycelicMemory/mycelicmemory/releases/download/v1.2.2/mycelicmemory-linux-arm64"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    else
      url "https://github.com/MycelicMemory/mycelicmemory/releases/download/v1.2.2/mycelicmemory-linux-x64"
      sha256 "PLACEHOLDER_LINUX_X64_SHA256"
    end
  end

  def install
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "mycelicmemory-macos-arm64" => "mycelicmemory"
      else
        bin.install "mycelicmemory-macos-x64" => "mycelicmemory"
      end
    else
      if Hardware::CPU.arm?
        bin.install "mycelicmemory-linux-arm64" => "mycelicmemory"
      else
        bin.install "mycelicmemory-linux-x64" => "mycelicmemory"
      end
    end
  end

  test do
    system "#{bin}/mycelicmemory", "--version"
    system "#{bin}/mycelicmemory", "--help"
  end
end
