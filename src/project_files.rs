use std::fs;
use toml_edit::{Document, Item};
use anyhow::{Result, Context};
use regex::Regex;
use serde_json;

pub enum ProjectType {
    Rust,
    Python,
    Cpp,
    Meson,
    PlatformIO,
}

pub struct ProjectFile {
    pub path: String,
    pub project_type: ProjectType,
}

impl ProjectFile {
    pub fn update_version(&self, new_version: &str) -> Result<()> {
        match self.project_type {
            ProjectType::Rust => self.update_cargo_toml(new_version),
            ProjectType::Python => self.update_pyproject_toml(new_version),
            ProjectType::Cpp => self.update_cmake_lists(new_version),
            ProjectType::Meson => self.update_meson_build(new_version),
            ProjectType::PlatformIO => self.update_platformio_project(new_version),
        }
    }

    fn update_cargo_toml(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read Cargo.toml")?;
        let mut doc = content.parse::<Document>()
            .context("Failed to parse Cargo.toml")?;
        
        doc["package"]["version"] = toml_edit::value(new_version);
        
        fs::write(&self.path, doc.to_string())
            .context("Failed to write Cargo.toml")
    }

    fn update_pyproject_toml(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read pyproject.toml")?;
        let mut doc = content.parse::<Document>()
            .context("Failed to parse pyproject.toml")?;
        
        if let Some(tool) = doc.get_mut("tool") {
            if let Some(Item::Table(poetry)) = tool.get_mut("poetry") {
                poetry["version"] = toml_edit::value(new_version);
            }
        }
        
        fs::write(&self.path, doc.to_string())
            .context("Failed to write pyproject.toml")
    }

    fn update_cmake_lists(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read CMakeLists.txt")?;
        
        // Extract major, minor, and patch versions
        let version_parts: Vec<&str> = new_version.split('.').collect();
        if version_parts.len() < 3 {
            return Err(anyhow::anyhow!("Invalid version format: {}", new_version));
        }
        
        let major = version_parts[0];
        let minor = version_parts[1];
        let patch = version_parts[2];
        
        // Update version-related variables in CMakeLists.txt
        let updated_content = update_cmake_version(&content, major, minor, patch)?;
        
        fs::write(&self.path, updated_content)
            .context("Failed to write CMakeLists.txt")
    }

    fn update_meson_build(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read meson.build")?;
        
        // Extract major, minor, and patch versions
        let version_parts: Vec<&str> = new_version.split('.').collect();
        if version_parts.len() < 3 {
            return Err(anyhow::anyhow!("Invalid version format: {}", new_version));
        }
        
        let major = version_parts[0];
        let minor = version_parts[1];
        let patch = version_parts[2];
        
        // Update version-related variables in meson.build
        let updated_content = update_meson_version(&content, new_version, major, minor, patch)?;
        
        fs::write(&self.path, updated_content)
            .context("Failed to write meson.build")
    }

    fn update_platformio_project(&self, new_version: &str) -> Result<()> {
        if self.path.ends_with("platformio.ini") {
            self.update_platformio_ini(new_version)
        } else if self.path.ends_with("library.json") {
            self.update_library_json(new_version)
        } else if self.path.ends_with("library.properties") {
            self.update_library_properties(new_version)
        } else {
            Err(anyhow::anyhow!("Unsupported PlatformIO file: {}", self.path))
        }
    }

    fn update_platformio_ini(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read platformio.ini")?;
        
        // platformio.ini uses a simple INI format
        let re = Regex::new(r"(version\s*=\s*)(.+)").unwrap();
        let updated_content = re.replace_all(&content, |caps: &regex::Captures| {
            format!("{}\"{}\"", &caps[1], new_version)
        }).to_string();
        
        fs::write(&self.path, updated_content)
            .context("Failed to write platformio.ini")
    }

    fn update_library_json(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read library.json")?;
        
        // Parse the JSON file
        let mut json: serde_json::Value = serde_json::from_str(&content)
            .context("Failed to parse library.json")?;
        
        // Update the version field
        if let Some(obj) = json.as_object_mut() {
            obj.insert("version".to_string(), serde_json::Value::String(new_version.to_string()));
        }
        
        // Serialize back to JSON
        let updated_content = serde_json::to_string_pretty(&json)
            .context("Failed to serialize library.json")?;
        
        fs::write(&self.path, updated_content)
            .context("Failed to write library.json")
    }

    fn update_library_properties(&self, new_version: &str) -> Result<()> {
        let content = fs::read_to_string(&self.path)
            .context("Failed to read library.properties")?;
        
        // library.properties uses a simple key=value format
        let re = Regex::new(r"(version\s*=\s*)(.+)").unwrap();
        let updated_content = re.replace_all(&content, |caps: &regex::Captures| {
            format!("{}{}", &caps[1], new_version)
        }).to_string();
        
        fs::write(&self.path, updated_content)
            .context("Failed to write library.properties")
    }
}

// Helper function to update version in CMakeLists.txt
fn update_cmake_version(content: &str, major: &str, minor: &str, patch: &str) -> Result<String> {
    let mut updated = String::new();
    let mut updated_project_version = false;
    let mut updated_version_vars = false;
    
    for line in content.lines() {
        if !updated_project_version && line.trim().starts_with("project(") && line.contains("VERSION") {
            // Update project version - format: project(ProjectName VERSION X.Y.Z)
            let parts: Vec<&str> = line.split("VERSION").collect();
            if parts.len() > 1 {
                let new_line = format!("{}VERSION {}.{}.{}", parts[0], major, minor, patch);
                updated.push_str(&new_line);
                updated_project_version = true;
            } else {
                updated.push_str(line);
            }
        } else if !updated_version_vars && 
                 (line.trim().starts_with("set(PROJECT_VERSION ") ||
                  line.trim().starts_with("set(CMAKE_PROJECT_VERSION ")) &&
                  !line.trim().contains("_MAJOR") &&
                  !line.trim().contains("_MINOR") &&
                  !line.trim().contains("_PATCH") {
            // Match set(PROJECT_VERSION X.Y.Z) or set(CMAKE_PROJECT_VERSION X.Y.Z)
            let new_line = format!("set(PROJECT_VERSION {}.{}.{})", major, minor, patch);
            updated.push_str(&new_line);
            updated_version_vars = true;
        } else if line.trim().starts_with("set(PROJECT_VERSION_MAJOR ") {
            updated.push_str(&format!("set(PROJECT_VERSION_MAJOR {})", major));
        } else if line.trim().starts_with("set(PROJECT_VERSION_MINOR ") {
            updated.push_str(&format!("set(PROJECT_VERSION_MINOR {})", minor));
        } else if line.trim().starts_with("set(PROJECT_VERSION_PATCH ") {
            updated.push_str(&format!("set(PROJECT_VERSION_PATCH {})", patch));
        } else {
            updated.push_str(line);
        }
        updated.push('\n');
    }
    
    Ok(updated)
}

// Helper function to update version in meson.build
fn update_meson_version(content: &str, full_version: &str, major: &str, minor: &str, patch: &str) -> Result<String> {
    let mut updated = String::new();
    
    // Handle line by line
    for line in content.lines() {
        // Match project version line
        if line.trim().starts_with("version : '") {
            let trimmed = line.trim();
            let indent = line[..(line.len() - trimmed.len())].to_string();
            let new_line = format!("{}version : '{}',", indent, full_version);
            updated.push_str(&new_line);
        }
        // Match version_major, version_minor, version_patch
        else if line.trim().starts_with("version_major = '") {
            let trimmed = line.trim();
            let indent = line[..(line.len() - trimmed.len())].to_string();
            let new_line = format!("{}version_major = '{}'", indent, major);
            updated.push_str(&new_line);
        }
        else if line.trim().starts_with("version_minor = '") {
            let trimmed = line.trim();
            let indent = line[..(line.len() - trimmed.len())].to_string();
            let new_line = format!("{}version_minor = '{}'", indent, minor);
            updated.push_str(&new_line);
        }
        else if line.trim().starts_with("version_patch = '") {
            let trimmed = line.trim();
            let indent = line[..(line.len() - trimmed.len())].to_string();
            let new_line = format!("{}version_patch = '{}'", indent, patch);
            updated.push_str(&new_line);
        }
        else {
            updated.push_str(line);
        }
        updated.push('\n');
    }
    
    Ok(updated)
} 