# kube-changejob Documentation

This directory contains the documentation for kube-changejob, published at [https://nusnewob.github.io/kube-changejob](https://nusnewob.github.io/kube-changejob).

## Documentation Structure

- **[index.md](index.md)** - Home page with overview and quick start ([view](https://nusnewob.github.io/kube-changejob/))
- **[installation.md](installation.md)** - Detailed installation instructions ([view](https://nusnewob.github.io/kube-changejob/installation))
- **[user-guide.md](user-guide.md)** - Complete usage guide with examples ([view](https://nusnewob.github.io/kube-changejob/user-guide))
- **[api-reference.md](api-reference.md)** - CRD specification and API details ([view](https://nusnewob.github.io/kube-changejob/api-reference))
- **[configuration.md](configuration.md)** - Controller configuration options ([view](https://nusnewob.github.io/kube-changejob/configuration))
- **[examples.md](examples.md)** - Real-world usage examples ([view](https://nusnewob.github.io/kube-changejob/examples))
- **[release.md](release.md)** - Release process and management ([view](https://nusnewob.github.io/kube-changejob/release))

## Building Docs Locally

To build and preview the documentation locally:

```bash
cd docs

# Install dependencies
bundle install

# Serve locally
bundle exec jekyll serve

# Open http://localhost:4000/kube-changejob in your browser
```

## GitHub Pages

The documentation is automatically built and deployed to GitHub Pages when changes are pushed to the `main` branch.

### Setup GitHub Pages

To enable GitHub Pages for this repository:

1. Go to repository Settings
2. Navigate to Pages section
3. Under "Build and deployment":
   - Source: GitHub Actions
4. The documentation will be available at: `https://nusnewob.github.io/kube-changejob`

## Theme

The documentation uses the [Just the Docs](https://just-the-docs.github.io/just-the-docs/) Jekyll theme, which provides:

- Clean, responsive design
- Built-in search functionality
- Easy navigation
- Code syntax highlighting
- Mobile-friendly layout

## Contributing to Documentation

When adding or modifying documentation:

1. Add Jekyll front matter to the top of each markdown file:

   ```yaml
   ---
   layout: default
   title: Page Title
   nav_order: 1
   ---
   ```

2. Use standard markdown syntax
3. Test locally before committing
4. Ensure links work correctly
5. Keep the table of contents up to date

## License

Documentation is licensed under Apache License 2.0, same as the project.
