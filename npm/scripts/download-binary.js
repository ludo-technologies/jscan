#!/usr/bin/env node

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const packageJson = require("../package.json");
const version = packageJson.version;

const platform = process.platform;
const arch = process.arch;

// Map Node.js platform/arch to Go build names
const platformMap = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

const platformName = platformMap[platform];
const archName = archMap[arch];

if (!platformName || !archName) {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const ext = platform === "win32" ? ".exe" : "";
const binaryName = `jscan-${platformName}-${archName}${ext}`;
const archiveName = `jscan_${version}_${platformName}_${archName}.tar.gz`;
const downloadUrl = `https://github.com/ludo-technologies/jscan/releases/download/v${version}/${archiveName}`;

const binDir = path.join(__dirname, "..", "bin");
const tempDir = path.join(__dirname, "..", ".tmp");
const binaryPath = path.join(binDir, binaryName);
const archivePath = path.join(tempDir, archiveName);

// Skip download if binary already exists
if (fs.existsSync(binaryPath)) {
  console.log(`jscan binary already exists at ${binaryPath}`);
  process.exit(0);
}

// Create directories if they don't exist
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}
if (!fs.existsSync(tempDir)) {
  fs.mkdirSync(tempDir, { recursive: true });
}

console.log(`Downloading jscan v${version} for ${platformName}-${archName}...`);
console.log(`URL: ${downloadUrl}`);

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);

    const request = (url) => {
      https
        .get(url, (response) => {
          // Handle redirects
          if (response.statusCode === 302 || response.statusCode === 301) {
            request(response.headers.location);
            return;
          }

          if (response.statusCode !== 200) {
            reject(new Error(`Failed to download: HTTP ${response.statusCode}`));
            return;
          }

          response.pipe(file);
          file.on("finish", () => {
            file.close();
            resolve();
          });
        })
        .on("error", (err) => {
          fs.unlink(dest, () => {});
          reject(err);
        });
    };

    request(url);
  });
}

async function main() {
  try {
    // Download the archive
    await download(downloadUrl, archivePath);
    console.log("Download complete.");

    // Extract the archive to temp directory
    console.log("Extracting...");
    execSync(`tar -xzf "${archivePath}" -C "${tempDir}"`, { stdio: "inherit" });

    // Move binary to bin directory with platform-specific name
    const extractedBinary = path.join(tempDir, `jscan${ext}`);
    if (fs.existsSync(extractedBinary)) {
      fs.renameSync(extractedBinary, binaryPath);
    }

    // Make it executable (Unix only)
    if (platform !== "win32") {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Clean up
    fs.unlinkSync(archivePath);
    fs.rmdirSync(tempDir);

    console.log(`jscan v${version} installed successfully!`);
  } catch (error) {
    console.error(`Failed to install jscan: ${error.message}`);
    console.error("");
    console.error("You can manually download the binary from:");
    console.error(`https://github.com/ludo-technologies/jscan/releases/tag/v${version}`);
    process.exit(1);
  }
}

main();
