# Makefile

# Set a help target for this makefile
help:
	@echo "Available targets:"
	@echo "  all                Build for all platforms (default)"
	@echo "  clean              Remove the build directory"
	@echo "  build-<OS>-<ARCH>  Build for a specific OS and Architecture"
	@echo ""
	@echo "Examples:"
	@echo "  make all VERSION=1.2.3"
	@echo "  make build-linux-amd64 VERSION=1.2.3"
	@echo "  make clean"

# Get the version from the command-line argument or set it to 0.0.1
VERSION ?= 0.0.1

# Set the name of the program
PROGRAM=CIDR-Sensei

# Set the directory where the program will be built
BUILD_DIR=build

# Set the list of all possible platforms
PLATFORMS=darwin/amd64 dragonfly/amd64 freebsd/amd64 freebsd/386 freebsd/arm64 linux/amd64 linux/386 linux/arm linux/arm64 linux/mips linux/mipsle linux/mips64 linux/mips64le linux/ppc64 linux/ppc64le linux/riscv64 linux/s390x openbsd/amd64 openbsd/386 plan9/amd64 plan9/386 solaris/amd64 windows/amd64 windows/386 windows/arm windows/arm64

# Set the name of the output file for each platform
define output_file
	$(BUILD_DIR)/$(PROGRAM)_$(VERSION)_$(1)_$(2)$(if $(findstring windows,$(1)),.exe,)
endef
OUTPUTS=$(foreach platform,$(PLATFORMS),$(call output_file,$(word 1,$(subst /, ,$(platform))),$(word 2,$(subst /, ,$(platform)))))

# Set the flags for each platform
define flags
	-o $(call output_file,$(1),$(2)) -ldflags="-s -w"
endef
FLAGS=$(foreach platform,$(PLATFORMS),$(call flags,$(word 1,$(subst /, ,$(platform))),$(word 2,$(subst /, ,$(platform)))))

# Define the build targets for each platform
define build_target
build-$(1)-$(2):
	@GOOS=$(1) GOARCH=$(2) go build $(call flags,$(1),$(2))
endef
$(foreach platform,$(PLATFORMS),$(eval $(call build_target,$(word 1,$(subst /, ,$(platform))),$(word 2,$(subst /, ,$(platform))))))

# Define the default target to build all platforms
all: $(foreach platform,$(PLATFORMS),build-$(word 1,$(subst /, ,$(platform)))-$(word 2,$(subst /, ,$(platform))))

# Define the clean target to remove the build directory
clean:
	@rm -rf $(BUILD_DIR)
