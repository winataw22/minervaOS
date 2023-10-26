#[macro_use]
extern crate anyhow;
#[macro_use]
extern crate log;

mod bootstrap;
mod config;
mod hub;
mod kparams;
mod workdir;
mod zfs;
mod zinit;

use anyhow::Result;
use config::Config;

fn app() -> Result<()> {
    let config = Config::current()?;

    let level = if config.debug {
        log::LevelFilter::Debug
    } else {
        log::LevelFilter::Info
    };

    simple_logger::SimpleLogger::new()
        .with_utc_timestamps()
        .with_level(level)
        .init()
        .unwrap();

    // configure available stage
    let stages: Vec<fn(cfg: &Config) -> Result<()>> = vec![
        // self update
        |cfg| -> Result<()> {
            if cfg.debug {
                // if debug is set, do not upgrade self.
                return Ok(());
            }
            bootstrap::update(cfg)
        },
        // install all system binaries
        |cfg| -> Result<()> { bootstrap::install(cfg) },
    ];

    let index = config.stage as usize - 1;

    if index >= stages.len() {
        bail!(
            "unknown stage '{}' only {} stage(s) are supported",
            config.stage,
            stages.len()
        );
    }

    info!("running stage {}/{}", config.stage, stages.len());
    stages[index](&config)?;

    // Why we run stages in different "processes" (hence using exec)
    // the point is that will allow the stages to do changes to the
    // bootstrap binary. It means an old image with an old version of
    // bootstrap will still be able to run latest code. Because always
    // the first stage is to update self.
    let next = config.stage as usize + 1;
    if next <= stages.len() {
        debug!("spawning stage: {}", next);
        let bin: Vec<String> = std::env::args().take(1).collect();
        let mut cmd = exec::Command::new(&bin[0]);
        let cmd = cmd.arg("-s").arg(format!("{}", next));
        let cmd = if config.debug { cmd.arg("-d") } else { cmd };

        //this call will never return unless something is wrong.
        bail!("{}", cmd.exec());
    }

    Ok(())
}

fn main() {
    let code = match app() {
        Ok(_) => 0,
        Err(err) => {
            eprintln!("{}", err);
            1
        }
    };

    std::process::exit(code);
}
