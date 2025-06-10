# go-starter

Use this project as a starter template for personal projects:

1. **Clone the base repository:**

    ```bash
    git clone https://github.com/raziel-aleman/go-starter.git
    ```

2. **Delete the existing Git history:**
   Navigate into the newly cloned directory and remove the `.git` directory. This step is crucial because it severs the connection to the original base repository's Git history.

    ```bash
    cd <newly_cloned_directory>
    rm -rf .git
    ```

3. **Initialize a new Git repository:**
   From within the cloned directory, initialize a fresh Git repository. This creates a brand-new, empty Git history for your new project.

    ```bash
    git init
    ```

4. **Add a new remote origin:**
   Connect your new local repository to a new remote repository on a platform like GitHub, Bitbucket, or others. Replace `<remote_URL>` with the URL of your new, empty repository.
    ```bash
    git remote add origin <remote_URL>
    ```

## MakeFile

Run build make command with tests

```bash
make all
```

Build the application

```bash
make build
```

Run the application

```bash
make run
```

Live reload the application:

```bash
make watch
```

Run the test suite:

```bash
make test
```

Clean up binary from the last build:

```bash
make clean
```
