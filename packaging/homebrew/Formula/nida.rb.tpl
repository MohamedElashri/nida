class Nida < Formula
  desc "Small Go static site generator for blogs and personal sites"
  homepage "https://github.com/MohamedElashri/nida"
  version "__VERSION__"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/MohamedElashri/nida/releases/download/__TAG__/nida_#{version}_darwin_arm64.tar.gz"
      sha256 "__DARWIN_ARM64_SHA__"
    else
      url "https://github.com/MohamedElashri/nida/releases/download/__TAG__/nida_#{version}_darwin_x86_64.tar.gz"
      sha256 "__DARWIN_X86_64_SHA__"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/MohamedElashri/nida/releases/download/__TAG__/nida_#{version}_linux_arm64.tar.gz"
      sha256 "__LINUX_ARM64_SHA__"
    else
      url "https://github.com/MohamedElashri/nida/releases/download/__TAG__/nida_#{version}_linux_x86_64.tar.gz"
      sha256 "__LINUX_X86_64_SHA__"
    end
  end

  def install
    bin.install "nida"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/nida version")
  end
end
