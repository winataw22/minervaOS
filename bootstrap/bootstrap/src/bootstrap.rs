use super::config;
use super::hub;
use super::kparams;
use super::workdir::WorkDir;
use super::zfs::Zfs;
use super::zinit;

use failure::Error;
use retry;

const FLIST_REPO: &str = "tf-zos";
const FLIST_INFO_FILE: &str = "/tmp/flist.info";
const FLIST_NAME_FILE: &str = "/tmp/flist.name";
const WORKDIR: &str = "/tmp/bootstrap";

type Result<T> = std::result::Result<T, Error>;

#[derive(Debug)]
enum RunMode {
    Prod,
    Test,
    Dev,
}

fn runmode() -> Result<RunMode> {
    let params = kparams::params()?;
    let mode = match params.get("runmode") {
        Some(mode) => match mode {
            Some(mode) => match mode.as_ref() {
                "prod" => RunMode::Prod,
                "dev" => RunMode::Dev,
                "test" => RunMode::Test,
                m => {
                    bail!("unknown runmode: {}", m);
                }
            },
            None => {
                //that's an error because runmode was
                //provided as a kernel argumet but with no
                //value
                bail!("missing runmode value");
            }
        },
        // runmode was not provided in cmdline
        // so default is prod
        None => RunMode::Prod,
    };

    Ok(mode)
}

fn boostrap_zos(mode: RunMode) -> Result<()> {
    let flist = match mode {
        RunMode::Prod => "zos:production:latest.flist",
        RunMode::Dev => "zos:development:latest.flist",
        RunMode::Test => "zos:testing:latest.flist",
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

    let result = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("download flist: {}", flist.name);

        // the entire point of this match is the
        // logging of the error.
        match flist.download("machine.flist") {
            Ok(ok) => Ok(ok),
            Err(err) => {
                error!("failed to download flist: {}", err);
                bail!("failed to download flist: {}", err);
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

    let fs = Zfs::mount("backend", "machine.flist", "root")?;
    debug!("zfs started, now copying all files");
    //copy everything to root
    match fs.copy("/") {
        Ok(_) => {}
        Err(err) => bail!("failed to copy files to rootfs: {}", err),
    };

    // we need to find all yaml files under /etc/zinit to start monitoring them
    let mut cfg = std::path::PathBuf::new();
    cfg.push(&fs);
    cfg.push("etc");
    cfg.push("zinit");
    let services = match std::fs::read_dir(&cfg) {
        Ok(services) => services,
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

/// bootstrap stage install and starts all zos daemons
pub fn bootstrap(_: &config::Config) -> Result<()> {
    let mode = runmode()?;
    debug!("runmode: {:?}", mode);
    let result = WorkDir::run(WORKDIR, || -> Result<()> {
        boostrap_zos(mode)?;
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

    let result = retry::retry(retry::delay::Exponential::from_millis(200), || {
        info!("download flist: {}", flist.name);

        // the entire point of this match is the
        // logging of the error.
        match flist.download("bootstrap.flist") {
            Ok(ok) => Ok(ok),
            Err(err) => {
                error!("failed to download flist: {}", err);
                bail!("failed to download flist: {}", err);
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

    let fs = Zfs::mount("backend", "bootstrap.flist", "root")?;
    debug!("zfs started, now copying all files");

    // this trick here to allow overriding
    // the current running bootstrap binary
    let bin: Vec<String> = std::env::args().take(1).collect();
    std::fs::rename(&bin[0], format!("{}.bak", &bin[0]))?;

    fs.copy("/")
}
