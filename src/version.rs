use semver::Version;
use std::path::Path;
use anyhow::Result;
use crate::project_files::{ProjectFile, ProjectType};
use std::fs;
use toml_edit::{Document, Item};
use regex::Regex;
use serde_json;

pub struct VersionManager {
    current_version: Version,
    pub project_files: Vec<ProjectFile>,
}

impl VersionManager {
    pub fn new() -> Self {
        // Default to 0.1.0 if no version is found
        VersionManager {
            current_version: Version::new(0, 1, 0),
            project_files: Vec::new(),
        }
    }

    pub fn bump_major(&mut self) -> &Version {
        self.current_version.major += 1;
        self.current_version.minor = 0;
        self.current_version.patch = 0;
        &self.current_version
    }

    pub fn bump_minor(&mut self) -> &Version {
        self.current_version.minor += 1;
        self.current_version.patch = 0;
        &self.current_version
    }

    pub fn bump_patch(&mut self) -> &Version {
        self.current_version.patch += 1;
        &self.current_version
    }

    pub fn detect_version_files(&mut self, project_root: &Path) -> Result<()> {
        let cargo_toml = project_root.join("Cargo.toml");
        let pyproject_toml = project_root.join("pyproject.toml");
        let cmake_lists = project_root.join("CMakeLists.txt");
        let meson_build = project_root.join("meson.build");
        let platformio_ini = project_root.join("platformio.ini");
        let library_json = project_root.join("library.json");
        let library_properties = project_root.join("library.properties");

        if cargo_toml.exists() {
            let content = fs::read_to_string(&cargo_toml)?;
            let doc = content.parse::<Document>()?;
            if let Some(version) = doc["package"]["version"].as_str() {
                self.current_version = Version::parse(version)?;
            }
            self.project_files.push(ProjectFile {
                path: cargo_toml.to_string_lossy().into_owned(),
                project_type: ProjectType::Rust,
            });
        }

        if pyproject_toml.exists() {
            let content = fs::read_to_string(&pyproject_toml)?;
            let doc = content.parse::<Document>()?;
            if let Some(tool) = doc.get("tool") {
                if let Some(Item::Table(poetry)) = tool.get("poetry") {
                    if let Some(version) = poetry["version"].as_str() {
                        self.current_version = Version::parse(version)?;
                    }
                }
            }
            self.project_files.push(ProjectFile {
                path: pyproject_toml.to_string_lossy().into_owned(),
                project_type: ProjectType::Python,
            });
        }

        if cmake_lists.exists() {
            let content = fs::read_to_string(&cmake_lists)?;
            if let Some(version) = extract_cmake_version(&content)? {
                self.current_version = version;
            }
            self.project_files.push(ProjectFile {
                path: cmake_lists.to_string_lossy().into_owned(),
                project_type: ProjectType::Cpp,
            });
        }

        if meson_build.exists() {
            let content = fs::read_to_string(&meson_build)?;
            if let Some(version) = extract_meson_version(&content)? {
                self.current_version = version;
            }
            self.project_files.push(ProjectFile {
                path: meson_build.to_string_lossy().into_owned(),
                project_type: ProjectType::Meson,
            });
        }

        // PlatformIO project detection
        if platformio_ini.exists() {
            let content = fs::read_to_string(&platformio_ini)?;
            if let Some(version) = extract_platformio_ini_version(&content)? {
                self.current_version = version;
            }
            self.project_files.push(ProjectFile {
                path: platformio_ini.to_string_lossy().into_owned(),
                project_type: ProjectType::PlatformIO,
            });
        }

        // PlatformIO library detection (library.json)
        if library_json.exists() {
            let content = fs::read_to_string(&library_json)?;
            if let Some(version) = extract_library_json_version(&content)? {
                self.current_version = version;
            }
            self.project_files.push(ProjectFile {
                path: library_json.to_string_lossy().into_owned(),
                project_type: ProjectType::PlatformIO,
            });
        }

        // PlatformIO library detection (library.properties)
        if library_properties.exists() {
            let content = fs::read_to_string(&library_properties)?;
            if let Some(version) = extract_library_properties_version(&content)? {
                self.current_version = version;
            }
            self.project_files.push(ProjectFile {
                path: library_properties.to_string_lossy().into_owned(),
                project_type: ProjectType::PlatformIO,
            });
        }

        Ok(())
    }

    pub fn update_all_versions(&self, new_version: &str) -> Result<()> {
        for project_file in &self.project_files {
            project_file.update_version(new_version)?;
        }
        Ok(())
    }

    pub fn get_current_version(&self) -> &Version {
        &self.current_version
    }
}

// Helper function to extract version from CMakeLists.txt
fn extract_cmake_version(content: &str) -> Result<Option<Version>> {
    // Try to find version in project() call first
    let project_re = Regex::new(r"project\s*\(\s*\w+\s+VERSION\s+(\d+)\.(\d+)\.(\d+)").unwrap();
    if let Some(caps) = project_re.captures(content) {
        let major: u64 = caps.get(1).unwrap().as_str().parse()?;
        let minor: u64 = caps.get(2).unwrap().as_str().parse()?;
        let patch: u64 = caps.get(3).unwrap().as_str().parse()?;
        return Ok(Some(Version::new(major, minor, patch)));
    }
    
    // Try to find version in set(PROJECT_VERSION) call
    let set_version_re = Regex::new(r"set\s*\(\s*(?:PROJECT|CMAKE_PROJECT)_VERSION\s+(\d+)\.(\d+)\.(\d+)").unwrap();
    if let Some(caps) = set_version_re.captures(content) {
        let major: u64 = caps.get(1).unwrap().as_str().parse()?;
        let minor: u64 = caps.get(2).unwrap().as_str().parse()?;
        let patch: u64 = caps.get(3).unwrap().as_str().parse()?;
        return Ok(Some(Version::new(major, minor, patch)));
    }
    
    // Try to find individual version components
    let mut major: Option<u64> = None;
    let mut minor: Option<u64> = None;
    let mut patch: Option<u64> = None;
    
    let major_re = Regex::new(r"set\s*\(\s*PROJECT_VERSION_MAJOR\s+(\d+)").unwrap();
    if let Some(caps) = major_re.captures(content) {
        major = Some(caps.get(1).unwrap().as_str().parse()?);
    }
    
    let minor_re = Regex::new(r"set\s*\(\s*PROJECT_VERSION_MINOR\s+(\d+)").unwrap();
    if let Some(caps) = minor_re.captures(content) {
        minor = Some(caps.get(1).unwrap().as_str().parse()?);
    }
    
    let patch_re = Regex::new(r"set\s*\(\s*PROJECT_VERSION_PATCH\s+(\d+)").unwrap();
    if let Some(caps) = patch_re.captures(content) {
        patch = Some(caps.get(1).unwrap().as_str().parse()?);
    }
    
    if let (Some(major), Some(minor), Some(patch)) = (major, minor, patch) {
        return Ok(Some(Version::new(major, minor, patch)));
    }
    
    Ok(None)
}

// Helper function to extract version from meson.build
fn extract_meson_version(content: &str) -> Result<Option<Version>> {
    // Try to find version in project() call first
    let project_re = Regex::new(r#"project\s*\(\s*['"][\w-]+['"](?:,\s*[^,)]+)*,\s*version\s*:\s*['"]([\d\.]+)['"]"#).unwrap();
    if let Some(caps) = project_re.captures(content) {
        let version_str = caps.get(1).unwrap().as_str();
        return parse_version_string(version_str);
    }
    
    // Try to find version variable declaration
    let version_var_re = Regex::new(r#"(version|project_version)\s*=\s*['"]([\d\.]+)['"]"#).unwrap();
    if let Some(caps) = version_var_re.captures(content) {
        let version_str = caps.get(2).unwrap().as_str();
        return parse_version_string(version_str);
    }
    
    // Try to find individual version components
    let mut major: Option<u64> = None;
    let mut minor: Option<u64> = None;
    let mut patch: Option<u64> = None;
    
    let major_re = Regex::new(r#"(version_major|major_version)\s*=\s*['"]?(\d+)['"]?"#).unwrap();
    if let Some(caps) = major_re.captures(content) {
        major = Some(caps.get(2).unwrap().as_str().parse()?);
    }
    
    let minor_re = Regex::new(r#"(version_minor|minor_version)\s*=\s*['"]?(\d+)['"]?"#).unwrap();
    if let Some(caps) = minor_re.captures(content) {
        minor = Some(caps.get(2).unwrap().as_str().parse()?);
    }
    
    let patch_re = Regex::new(r#"(version_patch|patch_version)\s*=\s*['"]?(\d+)['"]?"#).unwrap();
    if let Some(caps) = patch_re.captures(content) {
        patch = Some(caps.get(2).unwrap().as_str().parse()?);
    }
    
    if let (Some(major), Some(minor), Some(patch)) = (major, minor, patch) {
        return Ok(Some(Version::new(major, minor, patch)));
    }
    
    Ok(None)
}

// Helper function to parse version string
fn parse_version_string(version_str: &str) -> Result<Option<Version>> {
    let parts: Vec<&str> = version_str.split('.').collect();
    if parts.len() < 3 {
        return Ok(None);
    }
    
    let major: u64 = parts[0].parse()?;
    let minor: u64 = parts[1].parse()?;
    let patch: u64 = parts[2].parse()?;
    
    Ok(Some(Version::new(major, minor, patch)))
}

// Helper function to extract version from platformio.ini
fn extract_platformio_ini_version(content: &str) -> Result<Option<Version>> {
    let re = Regex::new(r#"version\s*=\s*["']?([\d\.]+)["']?"#).unwrap();
    if let Some(caps) = re.captures(content) {
        let version_str = caps.get(1).unwrap().as_str();
        return parse_version_string(version_str);
    }
    Ok(None)
}

// Helper function to extract version from library.json
fn extract_library_json_version(content: &str) -> Result<Option<Version>> {
    match serde_json::from_str::<serde_json::Value>(content) {
        Ok(json) => {
            if let Some(version_str) = json.get("version").and_then(|v| v.as_str()) {
                return parse_version_string(version_str);
            }
        },
        Err(_) => return Ok(None), // Handle parse error gracefully
    }
    Ok(None)
}

// Helper function to extract version from library.properties
fn extract_library_properties_version(content: &str) -> Result<Option<Version>> {
    let re = Regex::new(r"version\s*=\s*([\d\.]+)").unwrap();
    if let Some(caps) = re.captures(content) {
        let version_str = caps.get(1).unwrap().as_str();
        return parse_version_string(version_str);
    }
    Ok(None)
} 