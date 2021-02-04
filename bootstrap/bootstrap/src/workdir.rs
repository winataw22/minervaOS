use anyhow::Result;
use std::env;
use std::path::PathBuf;

pub struct WorkDir {
    path: PathBuf,
    old: PathBuf,
}

impl WorkDir {
    fn new<T>(path: T) -> Result<WorkDir>
    where
        T: Into<PathBuf>,
    {
        let path = path.into();
        let wd = WorkDir {
            path: path,
            old: env::current_dir()?,
        };
        debug!("creating: {:?}", wd.path);
        match std::fs::create_dir(&wd.path) {
            Err(e) => {
                if e.kind() != std::io::ErrorKind::AlreadyExists {
                    bail!("{}", e);
                }
            }
            _ => {}
        }
        debug!("mounting tmpfs");
        nix::mount::mount(
            Some("none"), // This should be set to None but for some reason the compiles complaines
            &wd.path,
            Some("tmpfs"),
            nix::mount::MsFlags::empty(),
            Some("size=512M"),
        )?;

        env::set_current_dir(&wd.path)?;
        Ok(wd)
    }

    pub fn run<T, F, O>(path: T, f: F) -> Result<O>
    where
        T: Into<PathBuf>,
        F: FnOnce() -> O,
    {
        let _wd = WorkDir::new(path)?;

        Ok(f())
    }
}

impl Drop for WorkDir {
    fn drop(&mut self) {
        match env::set_current_dir(&self.old) {
            Err(e) => {
                error!("failed change directory to: {}", e);
            }
            _ => {}
        }
        match nix::mount::umount2(&self.path, nix::mount::MntFlags::MNT_FORCE) {
            Err(e) => {
                error!("failed to unmount workdir: {}", e);
            }
            _ => {}
        }
        match std::fs::remove_dir_all(&self.path) {
            Err(e) => {
                error!("failed to delete workdir: {}", e);
            }
            _ => {}
        }
    }
}
