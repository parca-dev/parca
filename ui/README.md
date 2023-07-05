# Parca UI

This is a [Create React App](https://create-react-app.dev/) project that utilizes [Craco](https://github.com/gsoft-inc/craco) to modify and customize the app without 'ejecting'.

## Development

The React app requires an environment variable for the API endpoint so as to talk to the Parca backend. Create a file named `.env.local` in `packages/app/web/` to add the environment variable for the API endpoint.

```shell
REACT_APP_PUBLIC_API_ENDPOINT=http://localhost:7070
```

Then, start the Parca backend by running the command below. The `--cors-allowed-origins='*'` flag allows for enabling CORS headers on Parca.

```shell
./bin/parca --cors-allowed-origins='*'
```

Now the Parca backend will be running and available at `localhost:7070`.

Because we fetch the transpiled Typescript code from the shared `@parca` packages in the `shared/*` folder, we need to run one more command before we run the development for the React app. This command runs `tsc` in watch mode and also compiles Tailwind CSS for the affected packages.

```shell
yarn run watch
```

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
Run the following to build the `parca` binary with embedded assets.

```shell
make build
```

### Development workflow

> Before make sure all the tools you need are installed. The Linux users can simply run `make dev/setup`.

You can set up a cluster and all else you need by simply running:

```shell
make dev/up
```

For a simple local development setup, we use [Tilt](https://tilt.dev).

```shell
tilt up
```

## UI Feature Flags

We have a feature flag system that allows you to enable or disable features for a user browser. It's a naive implementation based on browser local storage.

### Usage

```js
import useUIFeatureFlag from '@parca/hooks';

const Header = () => {
  const isGreetingEnabled = useUIFeatureFlag('greeting');
  return (
    <div>
      <img src="/logo.png" alt="Logo" />
      {isGreetingEnabled ? <h1>Hello!!!</h1> : null}
    </div>
  );
};
```

For easy modification of the flag states, we added two utility query params that can be used to control the feature flag state: `enable-ui-flag` and `disable-ui-flag`.

For example, if you want to enable the greeting feature for a browser, you can load the following URL:

```text
http://localhost:3000/?enable-ui-flag=greeting
```

Likewise, if you would like to disable the greeting feature for a browser, you can load the following URL:

```text
http://localhost:3000/?disable-ui-flag=greeting
```

When the app loads with the above URL, the feature flags module will handle those and update the flag state accordingly.
Note: These 'enable' and 'disable' params work for setting one flag value at a time (rather than for example enabling "greeting" and another feature at the same time).

If you are interested in the implementation details, you can read the [source here](packages/shared/functions/src/useUIFeatureFlag/index.ts).

### Thanks

<a href="https://www.chromatic.com/"><img src="https://user-images.githubusercontent.com/321738/84662277-e3db4f80-af1b-11ea-88f5-91d67a5e59f6.png" width="153" height="30" alt="Chromatic" /></a>

Thanks to [Chromatic](https://www.chromatic.com/) for providing the visual testing platform that helps us review UI changes and catch visual regressions.
