# Parca UI

This is a [Create React App](https://create-react-app.dev/) project that utilizes [Craco](https://github.com/gsoft-inc/craco) to modify and customize the app without 'ejecting'.

## Development

The React app requires an environment variable for the API endpoint so as to talk to the Parca backend. Create a file named `.env.local` in `packages/app/web/` to add the environment variable for the API endpoint.

```
REACT_APP_PUBLIC_API_ENDPOINT=http://localhost:7070
```

Then, start the Parca backend by running the command below. The `--cors-allowed-origins='*'` flag allows for enabling CORS headers on Parca.

```shell
./bin/parca --cors-allowed-origins='*'
```

Now the Parca backend will be running and available at `localhost:7070`.

Finally, run the development server for the React app:

```shell
yarn workspace @parca/web dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the UI by modifying any of the components in the `ui/packages/app/web` directory. The app auto-updates as you edit the files.

## Build

To build the UI, you can use `Makefile` at the root of the project to run the following commands.

Run the following command to generate a production build of the React app:

```shell
make ui/build # yarn install && yarn build
```

We embed the artifacts (the production build and its static assets) into the final binary distribution.
See https://pkg.go.dev/embed for further details.

Run following to build the `parca` binary with embedded assets.

```shell
make build
```

### Development workflow

> Before make sure all the tools you need are installed. The Linux users can simply run `//env.sh`.

You can set up a cluster and all else you need by simply running:

```shell
make dev/up
```

For a simple local development setup we use [Tilt](https://tilt.dev).

```shell
tilt up
```
