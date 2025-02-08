use std::fs;
use toml_edit::{Document, Item};
use anyhow::{Result, Context};

pub enum ProjectType {
    Rust,
    Python,
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
} 