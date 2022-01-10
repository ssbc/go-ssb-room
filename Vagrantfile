Vagrant.configure("2") do |config|
  config.vm.box = "debian/bullseye64"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "3072"
    vb.cpus = 2
  end

  config.vm.network "forwarded_port", guest: 8899, host: 8899

  GO_VERSION = "1.17"
  config.vm.provision "shell" do |s|
    s.inline = <<~SHELL
      export DEBIAN_FRONTEND=noninteractive

      echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
      sudo apt-get update -q
      sudo apt-get install --no-install-recommends -y -q \
        goreleaser \
        git-core wget xz-utils openssl \
        nodejs npm \
        build-essential \
        crossbuild-essential-arm64 \
        crossbuild-essential-armhf

      sudo apt-get clean

      wget -q https://golang.org/dl/go#{GO_VERSION}.linux-amd64.tar.gz -O go#{GO_VERSION}.linux-amd64.tar.gz
      sudo tar -zxf go#{GO_VERSION}.linux-amd64.tar.gz -C /usr/local/
      echo "export PATH=/usr/local/go/bin:${PATH}" | sudo tee /etc/profile.d/go.sh
    SHELL
  end
end
