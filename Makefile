JAVA_PLUGIN_INSTALLED = $(cf plugins | grep -q)
JSTALL_JAR = dist/jstall-minimal.jar
# Set JSTALL_DEV=1 to pull the latest GitHub Actions build instead of the latest release
JSTALL_DEV ?=

all: install

$(JSTALL_JAR):
	mkdir -p dist
ifdef JSTALL_DEV
ifeq ($(JSTALL_DEV),1)
	gh run download -R parttimenerd/jstall -n jstall-minimal-jar --dir dist
else
	gh run download $(JSTALL_DEV) -R parttimenerd/jstall -n jstall-minimal-jar --dir dist
endif
else
	curl -sL -o $@ https://github.com/parttimenerd/jstall/releases/latest/download/jstall-minimal.jar
endif

download-jstall: $(JSTALL_JAR)

update-jstall:
	rm -f $(JSTALL_JAR)
	$(MAKE) download-jstall

.PHONY: update-jstall download-jstall

# When JSTALL_DEV=1, always re-download the jar (skip file existence check)
ifdef JSTALL_DEV
JSTALL_DEP = update-jstall
else
JSTALL_DEP = $(JSTALL_JAR)
endif

compile: $(JSTALL_DEP)
	go build -o build/cf-cli-java-plugin .

compile-all: $(JSTALL_DEP)
	GOOS=linux GOARCH=amd64 go build -o build/cf-cli-java-plugin-linux64 .
	GOOS=linux GOARCH=arm64 go build -o build/cf-cli-java-plugin-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o build/cf-cli-java-plugin-osx .
	GOOS=darwin GOARCH=arm64 go build -o build/cf-cli-java-plugin-osx-arm64 .
	GOOS=windows GOARCH=amd64 go build -o build/cf-cli-java-plugin-win64.exe .

clean:
	rm -r build

install: compile remove
	yes | cf install-plugin build/cf-cli-java-plugin

remove: $(objects)
ifeq ($(JAVA_PLUGIN_INSTALLED),)
	cf uninstall-plugin java || true
endif

vclean: remove clean