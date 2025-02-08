use clap::{Parser, ValueEnum};
use anyhow::Result;
use std::path::Path;
mod project_files;
mod version;
use version::VersionManager;
use crate::git::GitManager;
mod git;
use crate::changelog::ChangelogManager;
mod changelog;

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    /// Type of version bump to perform
    #[arg(value_enum)]
    bump_type: BumpType,

    /// Automatically push changes to remote
    #[arg(long)]
    push: bool,

    /// Create a GitHub release
    #[arg(long)]
    release: bool,
}

#[derive(Copy, Clone, PartialEq, Eq, ValueEnum)]
enum BumpType {
    Major,
    Minor,
    Patch,
    PreRelease,
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    let mut version_manager = VersionManager::new();
    let git_manager = GitManager::new()?;
    let changelog_manager = ChangelogManager::new()?;
    
    version_manager.detect_version_files(Path::new("."))?;
    let current_version = version_manager.get_current_version().to_string();
    
    let new_version = match cli.bump_type {
        BumpType::Major => version_manager.bump_major(),
        BumpType::Minor => version_manager.bump_minor(),
        BumpType::Patch => version_manager.bump_patch(),
        BumpType::PreRelease => version_manager.bump_patch(), // TODO: Implement pre-release
    };

    let version_string = new_version.to_string();
    println!("Updating version: {} -> {}", current_version, version_string);
    
    // Generate and preview changelog
    let changes = changelog_manager.generate_changes(Some(&current_version))?;
    println!("\nChangelog preview:\n{}", changes);
    
    println!("\nPress Enter to continue or Ctrl+C to cancel...");
    let mut input = String::new();
    std::io::stdin().read_line(&mut input)?;

    version_manager.update_all_versions(&version_string)?;
    changelog_manager.update_changelog(&version_string, &changes)?;
    
    // Git operations
    git_manager.commit_version_bump(&version_string)?;
    git_manager.create_tag(&version_string)?;
    
    println!("Successfully bumped version to {}", version_string);
    println!("Created git commit and tag v{}", version_string);
    println!("Updated CHANGELOG.md");

    if cli.push || cli.release {
        git_manager.push_changes(&version_string)?;
        println!("Pushed changes and tag to remote");
    }

    if cli.release {
        git_manager.create_github_release(&version_string, &changes)?;
        println!("Created GitHub release");
    }
    
    Ok(())
}
