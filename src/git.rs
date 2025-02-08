use anyhow::{Context, Result};
use git2::Repository;

pub struct GitManager {
    repo: Repository,
}

impl GitManager {
    pub fn new() -> Result<Self> {
        let repo = Repository::open(".")
            .context("Failed to open git repository")?;
        Ok(GitManager { repo })
    }

    pub fn commit_version_bump(&self, version: &str) -> Result<()> {
        let mut index = self.repo.index()?;
        index.add_all(["."], git2::IndexAddOption::DEFAULT, None)?;
        index.write()?;

        let tree_id = index.write_tree()?;
        let tree = self.repo.find_tree(tree_id)?;
        
        let head = self.repo.head()?;
        let parent_commit = self.repo.find_commit(head.target().unwrap())?;
        
        let signature = self.repo.signature()?;
        
        let message = format!("chore(release): bump version to {}", version);
        
        self.repo.commit(
            Some("HEAD"),
            &signature,
            &signature,
            &message,
            &tree,
            &[&parent_commit],
        )?;

        Ok(())
    }

    pub fn create_tag(&self, version: &str) -> Result<()> {
        let head = self.repo.head()?;
        let commit = self.repo.find_commit(head.target().unwrap())?;
        
        let signature = self.repo.signature()?;
        let tag_name = format!("v{}", version);
        let message = format!("Release version {}", version);

        self.repo.tag(
            &tag_name,
            commit.as_object(),
            &signature,
            &message,
            false,
        )?;

        Ok(())
    }

    pub fn push_changes(&self, version: &str) -> Result<()> {
        println!("Pushing changes to remote...");
        
        // Push both commit and tags
        let output = std::process::Command::new("git")
            .args(["push", "--follow-tags", "origin", "HEAD"])
            .output()
            .context("Failed to push changes")?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            println!("Push error: {}", stderr);
            anyhow::bail!("Failed to push changes to remote");
        }

        println!("Successfully pushed changes and tags to remote");
        Ok(())
    }

    pub fn create_github_release(&self, version: &str, changelog: &str) -> Result<()> {
        let tag_name = format!("v{}", version);
        
        println!("Creating GitHub release for tag {}", tag_name);
        
        // Create GitHub release using gh CLI with output capture
        let output = std::process::Command::new("gh")
            .args([
                "release",
                "create",
                &tag_name,
                "--title",
                &format!("Release {}", tag_name),
                "--notes",
                changelog,
                "--verify-tag",
                "--latest",
            ])
            .output()
            .context("Failed to execute gh CLI command")?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            let stdout = String::from_utf8_lossy(&output.stdout);
            println!("stdout: {}", stdout);
            println!("stderr: {}", stderr);
            anyhow::bail!("GitHub release creation failed");
        }

        println!("Release output: {}", String::from_utf8_lossy(&output.stdout));
        Ok(())
    }
} 