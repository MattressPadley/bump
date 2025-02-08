use anyhow::{Context, Result};
use git2::{Repository, Commit};
use std::fs;

pub struct ChangelogManager {
    repo: Repository,
}

impl ChangelogManager {
    pub fn new() -> Result<Self> {
        let repo = Repository::open(".")
            .context("Failed to open git repository")?;
        Ok(ChangelogManager { repo })
    }

    pub fn generate_changes(&self, from_version: Option<&str>) -> Result<String> {
        let mut changes = String::new();
        let head = self.repo.head()?.peel_to_commit()?;

        let from_commit = match from_version {
            Some(version) => {
                let tag_name = format!("v{}", version);
                match self.repo.find_reference(&format!("refs/tags/{}", tag_name)) {
                    Ok(reference) => reference.peel_to_commit()?,
                    Err(_) => {
                        println!("Warning: Previous version tag v{} not found, showing all commits", version);
                        self.get_first_commit()?
                    }
                }
            },
            None => {
                println!("No previous version specified, showing all commits");
                self.get_first_commit()?
            }
        };

        let mut revwalk = self.repo.revwalk()?;
        revwalk.push(head.id())?;
        revwalk.hide(from_commit.id())?;

        for oid in revwalk {
            let commit = self.repo.find_commit(oid?)?;
            let message = commit.message().unwrap_or("").trim();
            
            // Skip merge commits and version bump commits
            if message.starts_with("Merge") || message.contains("bump version") {
                continue;
            }

            // Format the commit message
            if let Some(formatted) = self.format_commit_message(message) {
                changes.push_str(&formatted);
                changes.push('\n');
            }
        }

        Ok(changes)
    }

    pub fn update_changelog(&self, version: &str, changes: &str) -> Result<()> {
        let changelog_dir = "docs";
        let changelog_path = format!("{}/CHANGELOG.md", changelog_dir);
        let date = chrono::Local::now().format("%Y-%m-%d");
        
        // Create docs directory if it doesn't exist
        std::fs::create_dir_all(changelog_dir)
            .context("Failed to create docs directory")?;
        
        let new_content = format!(
            "# {version} ({date})\n\n{changes}\n",
        );

        let existing_content = fs::read_to_string(&changelog_path)
            .unwrap_or_default();

        let updated_content = if existing_content.is_empty() {
            format!("# Changelog\n\n{new_content}")
        } else {
            // Find the position after the "# Changelog" header
            let pos = existing_content
                .find("# Changelog")
                .map(|i| i + "# Changelog".len())
                .unwrap_or(0);

            format!(
                "{}{}{}",
                &existing_content[..pos],
                "\n\n",
                new_content,
            )
        };

        fs::write(&changelog_path, updated_content)?;
        Ok(())
    }

    fn format_commit_message(&self, message: &str) -> Option<String> {
        // Skip empty messages
        if message.is_empty() {
            return None;
        }

        // Extract the first line
        let first_line = message.lines().next()?;
        
        // Format based on conventional commits
        if let Some(captures) = conventional_commit_regex().captures(first_line) {
            let type_ = captures.get(1)?.as_str();
            let scope = captures.get(2).map(|m| m.as_str());
            let description = captures.get(3)?.as_str();

            let formatted = match (type_, scope) {
                ("feat", Some(scope)) => format!("- ✨ **{}:** {}", scope, description),
                ("feat", None) => format!("- ✨ {}", description),
                ("fix", Some(scope)) => format!("- 🐛 **{}:** {}", scope, description),
                ("fix", None) => format!("- 🐛 {}", description),
                ("docs", _) => format!("- 📚 {}", description),
                ("style", _) => format!("- 💎 {}", description),
                ("refactor", _) => format!("- ♻️ {}", description),
                ("perf", _) => format!("- ⚡️ {}", description),
                ("test", _) => format!("- ✅ {}", description),
                ("build", _) => format!("- 📦 {}", description),
                ("ci", _) => format!("- 👷 {}", description),
                ("chore", _) => format!("- 🔧 {}", description),
                _ => format!("- {}", first_line),
            };
            Some(formatted)
        } else {
            Some(format!("- {}", first_line))
        }
    }

    fn get_first_commit(&self) -> Result<Commit> {
        let mut revwalk = self.repo.revwalk()?;
        revwalk.push_head()?;
        revwalk.set_sorting(git2::Sort::TIME)?;
        
        let first_id = revwalk.last()
            .context("No commits found")?
            .context("Failed to get commit id")?;
            
        self.repo.find_commit(first_id)
            .context("Failed to find first commit")
    }
}

fn conventional_commit_regex() -> regex::Regex {
    regex::Regex::new(r"^(\w+)(?:\(([^)]+)\))?: (.+)").unwrap()
} 