use semver::Version;
use std::path::Path;
use anyhow::Result;
use crate::project_files::{ProjectFile, ProjectType};
use std::fs;
use toml_edit::{Document, Item};

pub struct VersionManager {
    current_version: Version,
    project_files: Vec<ProjectFile>,
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