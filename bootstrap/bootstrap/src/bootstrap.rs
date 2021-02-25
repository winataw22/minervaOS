use super::config;
use super::hub;
use super::workdir::WorkDir;
use super::zfs::Zfs;
use super::zinit;
use anyhow::{Context, Result};
use config::{RunMode, Version};
use retry;

const FLIST_REPO: &str = "tf-zos";
const BIN_REPO: &str = "tf-zos-bins";
const FLIST_INFO_FILE: &str = "/tmp/flist.info";
const FLIST_NAME_FILE: &str = "/tmp/flist.name";
const WORKDIR: &str = "/tmp/bootstrap";

fn boostrap_zos(cfg: &config::Config) -> Result<()> {
    let flist = match &cfg.runmode {
        RunMode::Prod => match &cfg.version {
            Version::V2 => "zos:production:latest.flist",
            Version::V3 => "zos:production-3:latest.flist",
        },
        RunMode::Dev => match &cfg.version {
            Version::V2 => "zos:development:latest.flist",
            Version::V3 => "zos:development-3:latest.flist",
        },
        RunMode::Test => match &cfg.version {
            Version::V2 => "zos:testing:latest.flist",
            Version::V3 => "zos:testing-3:latest.flist",
        },
    };

    debug!("using flist: {}/{}", FLIST_REPO, flist);
    let repo = hub::Repo::new(FLIST_REPO);
    let flist = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("get flist info: {}", flist);
        let info = match repo.get(flist) {
            Ok(info) => info,
            Err(err) => {
                error!("failed to get info: {}", err);
                bail!("failed to get info: {}", err);
            }
        };

        Ok(info)
    });

    let flist = match flist {
        Ok(flist) => flist,
        Err(e) => bail!("failed to download flist: {:?}", e),
    };

    // write down boot info for other system components (like upgraded)
    flist.write(FLIST_INFO_FILE)?;
    std::fs::write(FLIST_NAME_FILE, format!("{}/{}", FLIST_REPO, flist.name))?;

    install_package(&flist)
}

/// bootstrap stage install and starts all zos daemons
pub fn bootstrap(cfg: &config::Config) -> Result<()> {
    debug!("runmode: {:?}", cfg.runmode);
    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        boostrap_zos(cfg)?;
        Ok(())
    })?;

    result
}

/// update stage make sure we are running latest
/// version of bootstrap
pub fn update(cfg: &config::Config) -> Result<()> {
    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        update_bootstrap(cfg.debug)?;
        Ok(())
    })?;

    result
}

// find the latest bootstrap binary on the hub. and make sure
// it's installed and available on the system, before starting
// the actual system bootstrap.
fn update_bootstrap(debug: bool) -> Result<()> {
    // we are running in a tmpfs workdir in this method
    let repo = hub::Repo::new("tf-autobuilder");
    let name = if debug {
        "bootstrap:development.flist"
    } else {
        "bootstrap:latest.flist"
    };

    let flist = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("get flist info: {}", name);
        //the full point of this match is the logging.
        let info = match repo.get(name) {
            Ok(info) => info,
            Err(err) => {
                error!("failed to get info: {}", err);
                bail!("failed to get info: {}", err);
            }
        };

        Ok(info)
    });

    let flist = match flist {
        Ok(flist) => flist,
        Err(e) => bail!("failed to download flist: {:?}", e),
    };

    // this trick here to allow overriding
    // the current running bootstrap binary
    let bin: Vec<String> = std::env::args().take(1).collect();
    std::fs::rename(&bin[0], format!("{}.bak", &bin[0]))?;

    install_package(&flist)
}

///install installs all binaries from the tf-zos-bins repo
pub fn install(cfg: &config::Config) -> Result<()> {
    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        install_packages(cfg)?;
        Ok(())
    })?;

    result
}

fn install_packages(cfg: &config::Config) -> Result<()> {
    let repo = match cfg.runmode {
        config::RunMode::Prod => BIN_REPO.to_owned(),
        config::RunMode::Dev => format!("{}.dev", BIN_REPO),
        config::RunMode::Test => format!("{}.test", BIN_REPO),
    };

    let client = hub::Repo::new(&repo);
    let packages = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("list packages in: {}", BIN_REPO);
        //the full point of this match is the logging.
        let packages = match client.list() {
            Ok(info) => info,
            Err(err) => {
                error!("failed to list repo '{}': {}", BIN_REPO, err);
                bail!("failed to list repo '{}': {}", BIN_REPO, err);
            }
        };

        Ok(packages)
    });

    let packages = match packages {
        Ok(packages) => packages,
        Err(err) => bail!("failed to list '{}': {:?}", BIN_REPO, err),
    };

    let mut map = std::collections::HashMap::new();
    for package in packages.iter() {
        match install_package(package) {
            Ok(_) => {}
            Err(err) => warn!("failed to install package '{}': {}", package.url, err),
        };

        map.insert(format!("{}/{}", repo, package.name), package.clone());
    }

    let output = std::fs::OpenOptions::new()
        .create(true)
        .write(true)
        .open("/tmp/bins.info")?;
    serde_json::to_writer(&output, &map)?;

    Ok(())
}

fn install_package(flist: &hub::Flist) -> Result<()> {
    let result = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("download flist: {}", flist.name);

        // the entire point of this match is the
        // logging of the error.
        match flist.download(&flist.name) {
            Ok(ok) => Ok(ok),
            Err(err) => {
                error!("failed to download flist '{}': {}", flist.url, err);
                bail!("failed to download flist '{}': {}", flist.url, err);
            }
        }
    });

    // I can't use the ? because error from retry
    // is not compatible with failure::Error for
    // some reason.
    match result {
        Err(err) => bail!("{:?}", err),
        _ => (),
    };

    let fs = Zfs::mount("backend", &flist.name, "root")?;
    debug!("zfs started, now copying all files");

    fs.copy("/").context("failed to copy files")?;

    debug!("starting services");
    run_all(&fs)
}

// run_all tries to run all services from an flist.
// it will still try to run all other services defined
// in the list of one or more failed. Returns error only
// if failed to read the zinit directory
fn run_all(fs: &Zfs) -> Result<()> {
    let mut cfg = std::path::PathBuf::new();
    cfg.push(fs);
    cfg.push("etc");
    cfg.push("zinit");
    let services = match std::fs::read_dir(&cfg) {
        Ok(services) => services,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(err) => bail!("failed to read directory '{:?}': {}", cfg, err),
    };

    for service in services {
        let service = service?;
        let path = service.path();

        if !path.is_file() {
            continue;
        }
        let name = match path.file_name() {
            Some(name) => match name.to_str() {
                Some(name) => name,
                None => {
                    warn!("failed to process name: {:?}", path);
                    continue;
                }
            },
            None => continue,
        };

        match name.rfind(".yaml") {
            None => continue,
            Some(idx) => {
                let service = &name[0..idx];
                match zinit::monitor(service) {
                    Ok(_) => {}
                    Err(err) => {
                        warn!("failed to monitor service '{}': {}", service, err);
                    }
                }
            }
        }
    }

    Ok(())
}
