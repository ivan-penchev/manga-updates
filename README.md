<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<h3 align="center"><img alt="logo" src="./logo.webp" style="width: 20vw;"></h3>

<<<<<<< HEAD
## Add a new series to check for updates
```sh
## make a folder inside the data folder with the title name
mkdir data/one-piece
## copy the title template inside as data.json
cp data/title.json.tpl data/one-piece/data.json 
```
Copy the slug from the url of the series e.g. https://manganel.me/manga/unexpected-accident has the slug of "unexpected-accident" inside the data.json file.

=======
>>>>>>> 2251135 (chore: updated README)
# Manga Updates

This project started as a simple idea: to keep tabs on my favorite manga series and get a heads-up (via Email) whenever a new chapter dropped. The initial thought was to manage everything through a GitHub repository, with updates triggered by a scheduled GitHub Actions Workflow.

But hey, there's nothing stopping you from running this locally on your machine! 

Plus, it's built with extensibility in mind, so feel free to tweak it or add your own twists.

## How to get Started

This project was originally designed to run within a GitHub repository, using a scheduled GitHub Actions workflow to automatically check for manga updates. The `example` directory showcases this initial concept.

### Directory Structure

The example demonstrates the expected directory structure for the project to work correctly with the GitHub workflow:

```
.
├── .github
│   └── workflows
│       └── send_email.yaml
└── data
    └── <manga-title>
        └── data.json
```

-   `.github/workflows/send_email.yaml`: This workflow file contains the logic to run the `manga-updates` binary on a schedule, check for new chapters, send email notifications, and commit any changes to the data files.
-   `data/<manga-title>/data.json`: Each manga series you want to track needs its own directory under `data`. The `data.json` file within each directory stores the manga's information.

### `data.json` File

The `data.json` file is crucial for tracking each manga series. Here's a breakdown of its fields:

```json
{
  "name": "Unexpected Accident",
  "shouldNotify": true,
  "lastUpdate": null,
  "slug": "unexpected-accident",
  "status": "",
  "latestChapter": null,
  "source": "manganel",
  "chapters": []
}
```

-   `name`: The human-readable name of the manga.
-   `shouldNotify`: Set to `true` if you want to receive email notifications for this manga.
-   `lastUpdate`: A timestamp indicating the last time the manga was checked for updates. This is updated automatically.
-   `slug`: The URL-friendly identifier of the manga on the source website (e.g., "unexpected-accident" for a manga located at `https://manganel.me/manga/unexpected-accident`).
-   `status`: The current status of the manga (e.g., "ongoing", "completed"). This is updated automatically.
-   `latestChapter`: The latest chapter number that has been detected. This is updated automatically.
-   `source`: The provider to use for checking updates. Currently supported providers are `manganel` and `mangadex`.
-   `chapters`: A list of chapters that have been detected. This is updated automatically.

### Setting Up a New Manga Series

To start monitoring a new manga series, you need to create a new `data.json` file for it. You can do this by copying the `data/title.json.tpl` file.

When setting up a new manga for the first time, you only need to fill in the following fields:

-   `name`: The human-readable name of the manga.
-   `source`: The provider to use for checking for updates. Currently, the supported providers are `manganel` and `mangadex`.
-   `slug`: This is the **most crucial field** for identifying the manga. It's a unique identifier from the manga source's URL.

#### Finding the `slug`

The `slug` is the part of the URL that directly points to the manga series. Here's how to find it for each supported provider:

-   **For `manganel`:**
    The slug is the last part of the URL. For example, for the manga at `https://manganel.me/manga/one-piece`, the slug is `one-piece`.

-   **For `mangadex`:**
    The slug is the UUID in the URL. For example, for the manga at `https://mangadex.org/title/a77742b6-363c-4310-9eca-2b7992395b3a/one-piece`, the slug is `a77742b6-363c-4310-9eca-2b7992395b3a`.

**It is essential that the `slug` is correct, otherwise the application will not be able to find the manga and check for updates.**

All other fields in the `data.json` file will be populated automatically by the application once it runs.

### `send_email.yaml` Workflow

The `send_email.yaml` workflow is the heart of the automated system. Before you can use it, you need to configure a few environment variables within the file:

-   `SMTP2GO_API_KEY`: Your API key for the SMTP2GO email service. It's highly recommended to store this as a secret in your GitHub repository.
-   `SMTP2GO_TEMPLATE_ID`: (Optional) The ID of the email template you want to use in SMTP2GO.
-   `NOTIFICATION_EMAIL_RECIPIENT`: The email address where you want to receive update notifications.
-   `NOTIFICATION_EMAIL_SENDER`: The email address that the notifications will be sent from.
-   `SERIES_DATAFOLDER`: The path to the directory where your manga data is stored (e.g., `./data`).

### How it Works

The `send_email.yaml` workflow performs the following steps:

1.  **Scheduled Trigger:** The workflow is configured to run at regular intervals (e.g., every 6 hours).
2.  **Checkout Code:** It checks out the latest version of your repository.
3.  **Run Manga Updates:** It downloads and runs the latest release of the `manga-updates` binary.
4.  **Check for Updates:** The binary reads the `data.json` files to know which manga to check. It then contacts the respective sources to see if new chapters are available.
5.  **Send Notifications:** If a new chapter is found, the application sends an email notification using the configured email provider.
6.  **Commit Changes:** If there are any changes to the `data.json` files (e.g., a new latest chapter is recorded), the workflow commits and pushes the changes back to your repository.

This setup provides a "set it and forget it" way to keep track of your favorite manga series.

## Components and Flow

Currently, the application is structured around several core components, each serving a specific purpose:

### Provider
These components are responsible for interacting with external manga sources to retrieve the latest chapter information for tracked series.
Currently we have:
- **MangaNel:** Fetches manga updates specifically from the MangaNel website. It leverages `chromedp` to interact with the website, extract information, and retrieve necessary cookies for API access.                                                                                       │
- **MangaDex:** Fetches manga updates from the MangaDex API, utilizing a dedicated client library (`mangodex`) for efficient data retrieval.

### Notifier 
- **SendGrid:** Sends email notifications via SendGrid.
- **SMTP2GO:** Sends email notifications via SMTP2GO.
- **Standard Output:** Prints notifications directly to the console (useful for testing and debugging).

### Store
- **Local files (JSON):** Manga series data is stored and managed in local JSON files within the `data` directory.

### Program Flow


```mermaid
<<<<<<< HEAD
graph TD
>>>>>>> 0cee209 (chore: updated README)
=======
graph LR
>>>>>>> fe6ee58 (chore: updated README)
    A[Start] --> B{Load Configuration}
    B --> C[Initialize Store]
    C --> D[Get Persisted Manga Series]
    D --> E{Initialize Notifier}
    E -- SendGrid/SMTP2GO/Stdout --> F[Initialize Provider Router]
    F -- MangaNel/MangaDex --> G[Initialize Update Checker Service]
    G --> H{Check For Updates}
    H --> I{For Each Manga Series}
    I --> J{Get Latest Version from Provider}
    J -- New Version Available? --> K{Notify via Notifier}
    K --> L[Update Persisted Data]
    L --> I
    I -- No More Series --> M[End]
    J -- No New Version --> I

```

## How to Run / How to Contribute

To set up and run the `manga-updates` application for development, follow these steps:

### 1. Clone the Repository

```bash
git clone https://github.com/ivan-penchev/manga-updates.git
cd manga-updates
```

### 2. Set Up Environment Variables

The application relies on several environment variables for configuration. Create a `local.env` file in the config folder. 
You can use the tpl available there.

```sh
cp ./config/local.tpl ./config/local.env
```

### 3. Run Chrome headless (for MangaNel Provider)

The MangaNel provider utilizes `chromedp` to interact with a headless Chrome instance. For local development or testing, you can run a headless Chrome browser in a Docker container.

 ```bash
    docker run -d -p 3000:3000 ghcr.io/browserless/chromium
 ```

This command starts a headless Chrome instance, exposing its DevTools Protocol.
The `manga-updates` application will automatically connect to this remote instance if `remoteURL` is set to `ws://127.0.0.1:3000`.

### 4. Run the Program



You have two primary ways to run the `manga-updates` application:



#### a) Using VS Code (Recommended for Development)



If you are using VS Code, you can leverage its integrated debugging and task running capabilities. Ensure your environment variables are set up in a `.env` file (as described in step 2) or directly in your VS Code launch configuration. VS Code will automatically pick up these variables when you run or debug the application.



#### b) Using `go run` (Command Line)



To run the application directly from your terminal using `go run`, you must ensure that all necessary environment variables are exposed in your shell session. You can do this by sourcing your `.env` file (if you created one) or by setting them individually.



Example of sourcing a `.env` file (assuming you named it `local.env` in the `config` folder as per step 2):



```bash

source config/local.env

go run cmd/manga-updates/main.go

```



Alternatively, set variables individually:



```bash

export NOTIFICATION_EMAIL_RECIPIENT="your_email@example.com"

export NOTIFICATION_EMAIL_SENDER="sender@example.com"

# ... other variables

go run cmd/manga-updates/main.go

```



The application will then check for updates for your configured manga series and send notifications if new chapters are found.