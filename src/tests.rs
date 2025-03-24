use std::fs;
use tempfile::tempdir;
use crate::version::VersionManager;
use crate::project_files::{ProjectFile, ProjectType};

// Rust project tests
#[test]
fn test_detect_cargo_version() {
    let dir = tempdir().unwrap();
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"

[dependencies]
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 1);
    assert_eq!(version.minor, 2);
    assert_eq!(version.patch, 3);
}

#[test]
fn test_update_cargo_version() {
    let dir = tempdir().unwrap();
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"

[dependencies]
"#).unwrap();

    let project_file = ProjectFile {
        path: cargo_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Rust,
    };
    
    project_file.update_version("2.3.4").unwrap();
    let content = fs::read_to_string(&cargo_path).unwrap();
    assert!(content.contains("version = \"2.3.4\""));
}

// Python project tests
#[test]
fn test_detect_python_version() {
    let dir = tempdir().unwrap();
    let pyproject_path = dir.path().join("pyproject.toml");
    fs::write(&pyproject_path, r#"
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"

[tool.poetry]
name = "test-project"
version = "2.3.4"
description = "Test project"
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 2);
    assert_eq!(version.minor, 3);
    assert_eq!(version.patch, 4);
}

#[test]
fn test_update_python_version() {
    let dir = tempdir().unwrap();
    let pyproject_path = dir.path().join("pyproject.toml");
    fs::write(&pyproject_path, r#"
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"

[tool.poetry]
name = "test-project"
version = "2.3.4"
description = "Test project"
"#).unwrap();

    let project_file = ProjectFile {
        path: pyproject_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Python,
    };
    
    project_file.update_version("3.4.5").unwrap();
    let content = fs::read_to_string(&pyproject_path).unwrap();
    assert!(content.contains("version = \"3.4.5\""));
}

// C++ CMake project tests
#[test]
fn test_detect_cmake_project_version() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject VERSION 3.4.5)

# Other cmake config
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 3);
    assert_eq!(version.minor, 4);
    assert_eq!(version.patch, 5);
}

#[test]
fn test_detect_cmake_set_version() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject)

set(PROJECT_VERSION 4.5.6)
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 4);
    assert_eq!(version.minor, 5);
    assert_eq!(version.patch, 6);
}

#[test]
fn test_detect_cmake_version_components() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject)

set(PROJECT_VERSION_MAJOR 5)
set(PROJECT_VERSION_MINOR 6)
set(PROJECT_VERSION_PATCH 7)
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 5);
    assert_eq!(version.minor, 6);
    assert_eq!(version.patch, 7);
}

#[test]
fn test_update_cmake_project_version() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject VERSION 3.4.5)

# Other cmake config
"#).unwrap();

    let project_file = ProjectFile {
        path: cmake_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Cpp,
    };
    
    project_file.update_version("6.7.8").unwrap();
    let content = fs::read_to_string(&cmake_path).unwrap();
    assert!(content.contains("project(TestProject VERSION 6.7.8"));
}

#[test]
fn test_update_cmake_set_version() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject)

set(PROJECT_VERSION 4.5.6)
"#).unwrap();

    let project_file = ProjectFile {
        path: cmake_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Cpp,
    };
    
    project_file.update_version("7.8.9").unwrap();
    let content = fs::read_to_string(&cmake_path).unwrap();
    assert!(content.contains("set(PROJECT_VERSION 7.8.9)"));
}

#[test]
fn test_update_cmake_version_components() {
    let dir = tempdir().unwrap();
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject)

set(PROJECT_VERSION_MAJOR 5)
set(PROJECT_VERSION_MINOR 6)
set(PROJECT_VERSION_PATCH 7)
"#).unwrap();

    let project_file = ProjectFile {
        path: cmake_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Cpp,
    };
    
    project_file.update_version("8.9.10").unwrap();
    let content = fs::read_to_string(&cmake_path).unwrap();
    println!("Generated content:\n{}", content);
    assert!(content.contains("set(PROJECT_VERSION_MAJOR 8)"));
    assert!(content.contains("set(PROJECT_VERSION_MINOR 9)"));
    assert!(content.contains("set(PROJECT_VERSION_PATCH 10)"));
}

// Meson project tests
#[test]
fn test_detect_meson_project_version() {
    let dir = tempdir().unwrap();
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp',
  version : '5.6.7',
  default_options : ['warning_level=3', 'cpp_std=c++17']
)
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 5);
    assert_eq!(version.minor, 6);
    assert_eq!(version.patch, 7);
}

#[test]
fn test_detect_meson_version_variable() {
    let dir = tempdir().unwrap();
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp')

version = '6.7.8'
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 6);
    assert_eq!(version.minor, 7);
    assert_eq!(version.patch, 8);
}

#[test]
fn test_detect_meson_version_components() {
    let dir = tempdir().unwrap();
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp')

version_major = '7'
version_minor = '8'
version_patch = '9'
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 7);
    assert_eq!(version.minor, 8);
    assert_eq!(version.patch, 9);
}

#[test]
fn test_update_meson_project_version() {
    let dir = tempdir().unwrap();
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp',
  version : '5.6.7',
  default_options : ['warning_level=3', 'cpp_std=c++17']
)
"#).unwrap();

    let project_file = ProjectFile {
        path: meson_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Meson,
    };
    
    project_file.update_version("8.9.10").unwrap();
    let content = fs::read_to_string(&meson_path).unwrap();
    assert!(content.contains("version : '8.9.10',"));
}

#[test]
fn test_update_meson_version_components() {
    let dir = tempdir().unwrap();
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp')

version_major = '7'
version_minor = '8'
version_patch = '9'
"#).unwrap();

    let project_file = ProjectFile {
        path: meson_path.to_string_lossy().into_owned(),
        project_type: ProjectType::Meson,
    };
    
    project_file.update_version("9.10.11").unwrap();
    let content = fs::read_to_string(&meson_path).unwrap();
    assert!(content.contains("version_major = '9'"));
    assert!(content.contains("version_minor = '10'"));
    assert!(content.contains("version_patch = '11'"));
}

// Version bump tests
#[test]
fn test_bump_major_version() {
    let mut version_manager = VersionManager::new();
    // Set initial version to 1.2.3
    let dir = tempdir().unwrap();
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"
"#).unwrap();
    
    version_manager.detect_version_files(dir.path()).unwrap();
    version_manager.bump_major();
    
    let new_version = version_manager.get_current_version();
    assert_eq!(new_version.major, 2);
    assert_eq!(new_version.minor, 0);
    assert_eq!(new_version.patch, 0);
}

#[test]
fn test_bump_minor_version() {
    let mut version_manager = VersionManager::new();
    // Set initial version to 1.2.3
    let dir = tempdir().unwrap();
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"
"#).unwrap();
    
    version_manager.detect_version_files(dir.path()).unwrap();
    version_manager.bump_minor();
    
    let new_version = version_manager.get_current_version();
    assert_eq!(new_version.major, 1);
    assert_eq!(new_version.minor, 3);
    assert_eq!(new_version.patch, 0);
}

#[test]
fn test_bump_patch_version() {
    let mut version_manager = VersionManager::new();
    // Set initial version to 1.2.3
    let dir = tempdir().unwrap();
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"
"#).unwrap();
    
    version_manager.detect_version_files(dir.path()).unwrap();
    version_manager.bump_patch();
    
    let new_version = version_manager.get_current_version();
    assert_eq!(new_version.major, 1);
    assert_eq!(new_version.minor, 2);
    assert_eq!(new_version.patch, 4);
}

// Multiple project types test
#[test]
fn test_multiple_project_types() {
    let dir = tempdir().unwrap();
    
    // Create Cargo.toml
    let cargo_path = dir.path().join("Cargo.toml");
    fs::write(&cargo_path, r#"
[package]
name = "test_project"
version = "1.2.3"
edition = "2021"
"#).unwrap();
    
    // Create CMakeLists.txt
    let cmake_path = dir.path().join("CMakeLists.txt");
    fs::write(&cmake_path, r#"
cmake_minimum_required(VERSION 3.10)
project(TestProject VERSION 1.2.3)
"#).unwrap();
    
    // Create meson.build
    let meson_path = dir.path().join("meson.build");
    fs::write(&meson_path, r#"
project('test_project', 'cpp',
  version : '1.2.3',
  default_options : ['warning_level=3', 'cpp_std=c++17']
)
"#).unwrap();
    
    // Create pyproject.toml
    let pyproject_path = dir.path().join("pyproject.toml");
    fs::write(&pyproject_path, r#"
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"

[tool.poetry]
name = "test-project"
version = "1.2.3"
description = "Test project"
"#).unwrap();
    
    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    // Verify all files were detected
    assert_eq!(version_manager.project_files.len(), 4);
    
    // Bump the version
    version_manager.bump_minor();
    let new_version = version_manager.get_current_version().to_string();
    
    // Update all project files
    version_manager.update_all_versions(&new_version).unwrap();
    
    // Check that each file was updated
    let cargo_content = fs::read_to_string(&cargo_path).unwrap();
    assert!(cargo_content.contains("version = \"1.3.0\""));
    
    let cmake_content = fs::read_to_string(&cmake_path).unwrap();
    assert!(cmake_content.contains("project(TestProject VERSION 1.3.0"));
    
    let meson_content = fs::read_to_string(&meson_path).unwrap();
    assert!(meson_content.contains("version : '1.3.0',"));
    
    let pyproject_content = fs::read_to_string(&pyproject_path).unwrap();
    assert!(pyproject_content.contains("version = \"1.3.0\""));
}

// PlatformIO tests
#[test]
fn test_detect_platformio_ini_version() {
    let dir = tempdir().unwrap();
    let platformio_ini_path = dir.path().join("platformio.ini");
    fs::write(&platformio_ini_path, r#"
[env:uno]
platform = atmelavr
board = uno
framework = arduino
version = "1.2.3"
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 1);
    assert_eq!(version.minor, 2);
    assert_eq!(version.patch, 3);
}

#[test]
fn test_update_platformio_ini_version() {
    let dir = tempdir().unwrap();
    let platformio_ini_path = dir.path().join("platformio.ini");
    fs::write(&platformio_ini_path, r#"
[env:uno]
platform = atmelavr
board = uno
framework = arduino
version = "1.2.3"
"#).unwrap();

    let project_file = ProjectFile {
        path: platformio_ini_path.to_string_lossy().into_owned(),
        project_type: ProjectType::PlatformIO,
    };
    
    project_file.update_version("2.3.4").unwrap();
    let content = fs::read_to_string(&platformio_ini_path).unwrap();
    assert!(content.contains("version = \"2.3.4\""));
}

#[test]
fn test_detect_library_json_version() {
    let dir = tempdir().unwrap();
    let library_json_path = dir.path().join("library.json");
    fs::write(&library_json_path, r#"
{
  "name": "TestLibrary",
  "version": "2.3.4",
  "description": "Test PlatformIO library",
  "keywords": "test, arduino",
  "repository": {
    "type": "git",
    "url": "https://github.com/example/test.git"
  },
  "authors": [
    {
      "name": "Test Author",
      "email": "test@example.com"
    }
  ],
  "frameworks": "*",
  "platforms": "*"
}
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 2);
    assert_eq!(version.minor, 3);
    assert_eq!(version.patch, 4);
}

#[test]
fn test_update_library_json_version() {
    let dir = tempdir().unwrap();
    let library_json_path = dir.path().join("library.json");
    fs::write(&library_json_path, r#"
{
  "name": "TestLibrary",
  "version": "2.3.4",
  "description": "Test PlatformIO library"
}
"#).unwrap();

    let project_file = ProjectFile {
        path: library_json_path.to_string_lossy().into_owned(),
        project_type: ProjectType::PlatformIO,
    };
    
    project_file.update_version("3.4.5").unwrap();
    let content = fs::read_to_string(&library_json_path).unwrap();
    assert!(content.contains("\"version\": \"3.4.5\""));
}

#[test]
fn test_detect_library_properties_version() {
    let dir = tempdir().unwrap();
    let properties_path = dir.path().join("library.properties");
    fs::write(&properties_path, r#"
name=TestLibrary
version=3.4.5
author=Test Author <test@example.com>
maintainer=Test Maintainer <maintainer@example.com>
sentence=A test library for PlatformIO.
paragraph=This is a longer description of the test library.
category=Other
url=https://github.com/example/test
architectures=*
"#).unwrap();

    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    let version = version_manager.get_current_version();
    assert_eq!(version.major, 3);
    assert_eq!(version.minor, 4);
    assert_eq!(version.patch, 5);
}

#[test]
fn test_update_library_properties_version() {
    let dir = tempdir().unwrap();
    let properties_path = dir.path().join("library.properties");
    fs::write(&properties_path, r#"
name=TestLibrary
version=3.4.5
author=Test Author <test@example.com>
"#).unwrap();

    let project_file = ProjectFile {
        path: properties_path.to_string_lossy().into_owned(),
        project_type: ProjectType::PlatformIO,
    };
    
    project_file.update_version("4.5.6").unwrap();
    let content = fs::read_to_string(&properties_path).unwrap();
    assert!(content.contains("version=4.5.6"));
}

#[test]
fn test_multiple_platformio_files() {
    let dir = tempdir().unwrap();
    
    // Create platformio.ini
    let platformio_ini_path = dir.path().join("platformio.ini");
    fs::write(&platformio_ini_path, r#"
[env:uno]
platform = atmelavr
board = uno
framework = arduino
version = "1.2.3"
"#).unwrap();
    
    // Create library.json
    let library_json_path = dir.path().join("library.json");
    fs::write(&library_json_path, r#"
{
  "name": "TestLibrary",
  "version": "1.2.3",
  "description": "Test PlatformIO library"
}
"#).unwrap();
    
    // Create library.properties
    let properties_path = dir.path().join("library.properties");
    fs::write(&properties_path, r#"
name=TestLibrary
version=1.2.3
author=Test Author <test@example.com>
"#).unwrap();
    
    let mut version_manager = VersionManager::new();
    version_manager.detect_version_files(dir.path()).unwrap();
    
    // Verify all files were detected
    assert_eq!(version_manager.project_files.len(), 3);
    
    // Bump the version
    version_manager.bump_minor();
    let new_version = version_manager.get_current_version().to_string();
    
    // Update all project files
    version_manager.update_all_versions(&new_version).unwrap();
    
    // Check that each file was updated
    let platformio_content = fs::read_to_string(&platformio_ini_path).unwrap();
    assert!(platformio_content.contains("version = \"1.3.0\""));
    
    let library_json_content = fs::read_to_string(&library_json_path).unwrap();
    assert!(library_json_content.contains("\"version\": \"1.3.0\""));
    
    let properties_content = fs::read_to_string(&properties_path).unwrap();
    assert!(properties_content.contains("version=1.3.0"));
} 