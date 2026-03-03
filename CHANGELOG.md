## [0.1.1](https://github.com/nusnewob/kube-changejob/compare/v0.1.0..v0.1.1) - 2026-02-01

### 🐛 Bug Fixes

- Fix test-chart pipeline - ([c27bb2f](https://github.com/nusnewob/kube-changejob/commit/c27bb2f24e40df0a97914a8aac884cb9022e5e9e))
- Use git-cliff to generate changelog for releasee pipeline - ([797b8fb](https://github.com/nusnewob/kube-changejob/commit/797b8fbe511fa99f1ae09030bfa1859a21404515))
- Fix typos - ([f290967](https://github.com/nusnewob/kube-changejob/commit/f2909670e83c37c5c6fe20cca1cc05d8782194c7))
- Cleanup github release pipeline and changelog - ([2bc4918](https://github.com/nusnewob/kube-changejob/commit/2bc4918966ebf56ff0372ff6c960f28a83b45323))

### ⚙️ Miscellaneous Tasks

- Add typos check to pipelines - ([0b60549](https://github.com/nusnewob/kube-changejob/commit/0b60549a584a3c9f7886b752566032e29173ba13))
- Remove scripts not used - ([e3f2dbf](https://github.com/nusnewob/kube-changejob/commit/e3f2dbf76dd48630265deaa0ea46c64188188476))
- Update dependabot config - ([1584e1a](https://github.com/nusnewob/kube-changejob/commit/1584e1acfac32ae24bac4fc19a375af7ff58c95c))
- Update dependabot config - ([54afe06](https://github.com/nusnewob/kube-changejob/commit/54afe0662afd1af90dc020b005e143ff1c82c5cd))
- Update CHANGELOG.md for release v0.1.0 - ([d20a0ac](https://github.com/nusnewob/kube-changejob/commit/d20a0acd398085b68169744bf89d666cde451db9))


## [0.1.0](https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.3..v0.1.0) - 2026-01-29

### 🐛 Bug Fixes

- *(controller)* Improve logic, robustness, and functionality ([#28](https://github.com/nusnewob/kube-changejob/issues/28)) - ([8e32e07](https://github.com/nusnewob/kube-changejob/commit/8e32e0729d9502cf3bdad52d5c3d85316cbffcd0))
- *(controller)* Improve logic, robustness, and functionality - ([9ed8fb2](https://github.com/nusnewob/kube-changejob/commit/9ed8fb2144c939b4e2d6477d4dd6805001d55fe4))
- Preallocating array to fix golangci-lint error - ([69cf445](https://github.com/nusnewob/kube-changejob/commit/69cf4452795666cd23b6cba9a07b9204090d3eb8))
- Kubebuilder upgrade confilicts - ([eb9bdfe](https://github.com/nusnewob/kube-changejob/commit/eb9bdfe898545fcddcbe8de52dfa7af6d727709f))
- Codecov config ([#30](https://github.com/nusnewob/kube-changejob/issues/30)) - ([b32c914](https://github.com/nusnewob/kube-changejob/commit/b32c9142fab20c157d11b09ecd51363770730011))
- Codecov config - ([7ebc7b2](https://github.com/nusnewob/kube-changejob/commit/7ebc7b2baaf37f4bef6bfbd296e7ae4ab0893a1a))
- Indexing namespace and cluster scoped resources - ([db99b47](https://github.com/nusnewob/kube-changejob/commit/db99b476b1f3d0bd71fd5a99da8339c21a7f18ba))
- Fix release pipeline to update CHANGELOG.md - ([3ca9a7a](https://github.com/nusnewob/kube-changejob/commit/3ca9a7a8e9bd1a948d929ff7d55de2fff665e8cf))

### 🧪 Testing

- Add test for cluster-scoped resources - ([5c8dd91](https://github.com/nusnewob/kube-changejob/commit/5c8dd918670b9b3c12f9c7ac05a0a215a386bdb8))

### ⚙️ Miscellaneous Tasks

- Bump version to v0.1.0 - ([7e9c2de](https://github.com/nusnewob/kube-changejob/commit/7e9c2ded24841a5ed7ce11f716fa618e66c18568))

## New Contributors ❤️

* @github-actions[bot] made their first contribution
* @google-labs-jules[bot] made their first contribution in [#29](https://github.com/nusnewob/kube-changejob/pull/29)

## [0.1.0-alpha.3](https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.2..v0.1.0-alpha.3) - 2026-01-15

### 🐛 Bug Fixes

- Bug 2 unsafe json.Marshal ordering - ([dc08b79](https://github.com/nusnewob/kube-changejob/commit/dc08b7986592d7d6c873c146f7df73fe91eb599a))
- Always update status during controller reconcile ([#20](https://github.com/nusnewob/kube-changejob/issues/20)) - ([0332858](https://github.com/nusnewob/kube-changejob/commit/0332858d3e5a4026ba14dc4fbd5acfd8e65815df))
- Always update status during controller reconcile - ([9167811](https://github.com/nusnewob/kube-changejob/commit/91678114dfa270d6bc6effc018e6ad9b6bcc3fff))

### 🧪 Testing

- Add tests to improve test coverage - ([454fc68](https://github.com/nusnewob/kube-changejob/commit/454fc682444378b336a0df740340e28cd5401533))

### ⚙️ Miscellaneous Tasks

- Cleanup and improve github pipelines ([#25](https://github.com/nusnewob/kube-changejob/issues/25)) - ([40ca51f](https://github.com/nusnewob/kube-changejob/commit/40ca51f849d76cf6140a8970804b14fe173a9674))
- Cleanup and improve github pipelines - ([931d920](https://github.com/nusnewob/kube-changejob/commit/931d920918de3a366eac88a1ebf491f264c7f83a))
- Remove duplicate docs - ([a7fad88](https://github.com/nusnewob/kube-changejob/commit/a7fad8813cdf077488b2b0b5e508338f3f07efb1))
- Add webhook validation for spec.cooldown value - ([c218484](https://github.com/nusnewob/kube-changejob/commit/c218484aa8eecff9b3335394288026e4e384bde7))
- Reduce unnecessary rbac permission for watched resources ([#19](https://github.com/nusnewob/kube-changejob/issues/19)) - ([0997600](https://github.com/nusnewob/kube-changejob/commit/09976003fa7f4838b37ed3dc6dd7be03e1e25366))
- Reduce unnecessary rbac permission for watched resources - ([a598dab](https://github.com/nusnewob/kube-changejob/commit/a598daba8113b824e98b29429425418d0a184732))


## [0.1.0-alpha.2](https://github.com/nusnewob/kube-changejob/compare/v0.1.0-alpha.1..v0.1.0-alpha.2) - 2025-12-30

### 🐛 Bug Fixes

- Fix controller logic and premature returns - ([b11fa65](https://github.com/nusnewob/kube-changejob/commit/b11fa6502a32a9f77db671713212249e312ca85c))

### 🚜 Refactor

- Consolidate update status logic and function - ([107e066](https://github.com/nusnewob/kube-changejob/commit/107e066ffd9a7c738d35ac15c0f433e5d934e779))
- Cleanup controller utility functions - ([2a590ab](https://github.com/nusnewob/kube-changejob/commit/2a590ab82cede158c348c03bc99a0806ea1759f1))

### 📚 Documentation

- Update readme.md - ([84501aa](https://github.com/nusnewob/kube-changejob/commit/84501aa3736dfb947581a310145be3b8a66efe6f))

### 🧪 Testing

- *(refactor)* Refactor e2e tests to use k8s go client instead of kubectl, improve speed and efficiency - ([9b43f01](https://github.com/nusnewob/kube-changejob/commit/9b43f01ac53e0643c57045a69f64206e32ee5f92))
- Add codecov.yml - ([29f9e2d](https://github.com/nusnewob/kube-changejob/commit/29f9e2dd8e5dc2d2d2fc187950128284841346dd))
- Update tests to cover more test cases - ([209be8f](https://github.com/nusnewob/kube-changejob/commit/209be8f53f60b205eea5debb021baf27a6933e2e))

### ⚙️ Miscellaneous Tasks

- Update github action triggers allow manual run - ([aa16588](https://github.com/nusnewob/kube-changejob/commit/aa1658848d3e0f730f7006d97d7618ca83864942))
- Update codecov.yml - ([b7dda05](https://github.com/nusnewob/kube-changejob/commit/b7dda05e9ee7d6d29687e80b34448ffecd5b3e90))
- Improve controller logic, add additional validation - ([16cdce6](https://github.com/nusnewob/kube-changejob/commit/16cdce6e76e264a23413838a4d5fa1a37488ec4a))
- Remove unnecessary code - ([7a23c83](https://github.com/nusnewob/kube-changejob/commit/7a23c838d1f17581e015c6f3177dc094fcbc45e3))
- Configurable RBAC for watched resources in helm - ([ec701e5](https://github.com/nusnewob/kube-changejob/commit/ec701e5e7fe37fb6acae23ed174181ef204664d1))
- Cleanup - ([2ad82ab](https://github.com/nusnewob/kube-changejob/commit/2ad82ab4fca3758adbe55041aeafdc6b67e78883))


## [0.1.0-alpha.1] - 2025-12-23

### ⛰️  Features

- *(wip)* Add controller and webhook logic + update CRD + etc - ([13e8b00](https://github.com/nusnewob/kube-changejob/commit/13e8b001631f528686bb1881c955d0a28b39811b))
- Add logging options for controller cmd - ([81d7ba8](https://github.com/nusnewob/kube-changejob/commit/81d7ba80f0bffd78c918ae467e56efc246834a2b))
- Configurable cmd/envar PollInterval - ([a743109](https://github.com/nusnewob/kube-changejob/commit/a7431098a75f3e45bebca3c419d6ab011b68b402))
- Add max job history - ([0b67fef](https://github.com/nusnewob/kube-changejob/commit/0b67fef64d73a7ced924cc972d31ca34d19a3312))

### 🐛 Bug Fixes

- *(docs)* Fix docs links - ([24bc4c6](https://github.com/nusnewob/kube-changejob/commit/24bc4c60dc5e88d3e171d3313f4d95aae8bbc16f))
- Fix release pipeline - ([05bd8c7](https://github.com/nusnewob/kube-changejob/commit/05bd8c75f02eba51c23d5e27f58d473eb0cf89bf))
- Fix github action to sync docs to wiki - ([9e0a35f](https://github.com/nusnewob/kube-changejob/commit/9e0a35f45d4f50854500b2350b8e1254a6002eca))
- Github actions for docs - ([55dbcc2](https://github.com/nusnewob/kube-changejob/commit/55dbcc24e0397651a507a3b93daca3cb296aab74))
- Fix helm chart test - ([38e40cc](https://github.com/nusnewob/kube-changejob/commit/38e40cc31f8704431562623c2ef59702cc86ded5))
- Docs and github actions - ([51ebb76](https://github.com/nusnewob/kube-changejob/commit/51ebb76cc5fd26c53cf08bc2e1fc445376af7f13))
- Linting error fix - ([020ab6c](https://github.com/nusnewob/kube-changejob/commit/020ab6c7545190de67a9d86cf24ec252788e14db))
- Fix updating LastJobStatus during controller reconcile - ([ebec3f3](https://github.com/nusnewob/kube-changejob/commit/ebec3f3a4a44102f1742f71001c3435c061e24fb))
- Fix when watching whole object (fields=[]) cause triggering jobs repeatedly - ([670c927](https://github.com/nusnewob/kube-changejob/commit/670c92732999981782e9304f2f9c4cea46c75144))
- Fix logic for new change triggered job shouldn't trigger a job - ([abd770b](https://github.com/nusnewob/kube-changejob/commit/abd770b38dc55a54e15a333706e6869c407fd3bf))
- Fix e2e tests - ([feb1a83](https://github.com/nusnewob/kube-changejob/commit/feb1a83d8cf02ae823577ed6475853ffe2208c0b))
- Fix controller + e2e test - ([39f6998](https://github.com/nusnewob/kube-changejob/commit/39f699841f246563b9beec12a7f6fc2ca84e0a28))
- Go mod tidy and cleanup imports - ([671613f](https://github.com/nusnewob/kube-changejob/commit/671613fcb4a0313174f205592263293d977a51fb))

### 🚜 Refactor

- Update logging - ([e222eab](https://github.com/nusnewob/kube-changejob/commit/e222eab6c2b2edae27c19c3c08396348dfe160e7))
- Update CRD for webhook validation and its tests - ([1ac098c](https://github.com/nusnewob/kube-changejob/commit/1ac098ca107ee40b75d7d8ca8b5173911ac8780f))
- Use cheaper .dev domain instead of .io - ([2b531d7](https://github.com/nusnewob/kube-changejob/commit/2b531d7fc81bb868b7989e287e45cdcf4b2229e3))

### 📚 Documentation

- Update docs to include release guide - ([c17b456](https://github.com/nusnewob/kube-changejob/commit/c17b45663afa8983fb022db272a1156eaff69539))
- Add github page docs and readme - ([db80441](https://github.com/nusnewob/kube-changejob/commit/db8044198f8678e37b00c380f879cbb8bffc9ec8))
- Add github issue/pr templates + community docs - ([e7f197d](https://github.com/nusnewob/kube-changejob/commit/e7f197d8e981e586a7f7c36516c3c8ea176b511a))

### 🧪 Testing

- Add additional tests - ([6f0671e](https://github.com/nusnewob/kube-changejob/commit/6f0671e88f1fab82dfdf6d753cb41296ce30c137))
- Add tests for cmd logging options - ([a4b433d](https://github.com/nusnewob/kube-changejob/commit/a4b433d2f37fdfd5a979d3bbca87a3eab1cdfb0f))
- Add tests for configurable PollInterval - ([6bbe4ca](https://github.com/nusnewob/kube-changejob/commit/6bbe4ca96e6ca0bbda71322b0ba7ca4b02d78d83))
- Add test for max job history - ([92369c5](https://github.com/nusnewob/kube-changejob/commit/92369c5b9fede4a84f57307dfbce9e08d21b0d09))
- Fix e2e tests + add new test scenarios - ([b6f8546](https://github.com/nusnewob/kube-changejob/commit/b6f8546c02bf0fa3e84e603e6891ecb1f52519e2))
- Update e2e tests - ([7de1d9d](https://github.com/nusnewob/kube-changejob/commit/7de1d9d7311769418be23b6d19cd542d7acf1dd4))
- Update unit tests for controller and webhook - ([545ce77](https://github.com/nusnewob/kube-changejob/commit/545ce775ca1eb6ae6b0cef14d8095090db8a8178))

### ⚙️ Miscellaneous Tasks

- *(refactor)* Add validation to webhook + refactor util func - ([c5bd408](https://github.com/nusnewob/kube-changejob/commit/c5bd40875ab99a36a8d81bfe00db7d85ba120442))
- Bump version to v0.1.0-alpha.1 - ([0008f0a](https://github.com/nusnewob/kube-changejob/commit/0008f0ad848aecf882b98d5ff7872caa343d1359))
- Add github action for release - ([487cdfc](https://github.com/nusnewob/kube-changejob/commit/487cdfc72da0dedfc078cae7594cbf9b1027dff3))
- Add github action for wiki pages - ([40d7ec8](https://github.com/nusnewob/kube-changejob/commit/40d7ec85101545f567cf9255ccdc108d32be2068))
- Update helm chart - ([e7bc977](https://github.com/nusnewob/kube-changejob/commit/e7bc97757b1e6bbadf6e77de8cf6de1ac8bb20eb))
- Add shortname for ChangeTriggeredJob CRD - ([8c9171e](https://github.com/nusnewob/kube-changejob/commit/8c9171ec83bba9beb4319c379bfea073013bc7f5))
- Add codecov github action - ([1112bb1](https://github.com/nusnewob/kube-changejob/commit/1112bb126c25bacc437c708ea2c3536cd3e5b2f2))
- Update github action e2e tests only for PRs - ([72828ab](https://github.com/nusnewob/kube-changejob/commit/72828ab430eca4a1ba277e23475cde08a8e542c8))
- Switch license to Apache 2.0 - ([4a59650](https://github.com/nusnewob/kube-changejob/commit/4a59650fc161ca728608214237b928acd5ba90d7))
- Upgrade github action versions - ([8fa78b8](https://github.com/nusnewob/kube-changejob/commit/8fa78b89a94946bc8f108b88370398aa3e291a7b))
- Minor improv on controller reconcile logic - ([b44785f](https://github.com/nusnewob/kube-changejob/commit/b44785f97f0c96865f35fa76f16aa09a0af0a215))
- Update misc resources - ([3dc3ff7](https://github.com/nusnewob/kube-changejob/commit/3dc3ff75e8158a4d022cd571ff604ac78a0ab314))
- Update CRD - ([fade370](https://github.com/nusnewob/kube-changejob/commit/fade3709491d869c3f324e0e1cbfa16b217c36b9))
- Define basic API structure - ([9ff628b](https://github.com/nusnewob/kube-changejob/commit/9ff628b66364919110bf3b56d3552465dd39949b))
- Kubebuilder create webhook - ([fa58306](https://github.com/nusnewob/kube-changejob/commit/fa5830690c9a495485171bc65964a30e38edbc05))
- Kubebuilder create api - ([903b2a1](https://github.com/nusnewob/kube-changejob/commit/903b2a152bb3969d265b0e4739875c66ab4cd29e))
- Kubebuilder init - ([33d4e4c](https://github.com/nusnewob/kube-changejob/commit/33d4e4cbf297ff90ed10afac707d9eaf9cd88784))

### Cleanup

- Remove cert-manager tests not needed - ([c11101c](https://github.com/nusnewob/kube-changejob/commit/c11101ce868d4e1786c4edf3be645be9f3af83fa))

### Wip

- *(tests)* Update controller test - ([4d14f74](https://github.com/nusnewob/kube-changejob/commit/4d14f74300f594f489b1ca915d9149d462156ab7))
- Improve controller logics and funcs - ([44e566c](https://github.com/nusnewob/kube-changejob/commit/44e566ca0e56afb8589560b3dd9bdf8d5a361574))
- Add cooldown delay - ([61f4844](https://github.com/nusnewob/kube-changejob/commit/61f4844d0f042c7945d7b25f5daadfafe95407a7))
- Add periodic polling in controller - ([7d1d2c8](https://github.com/nusnewob/kube-changejob/commit/7d1d2c8ddf9bc4b3e798b76d6deb6af58293b31c))
- Controller + misc func - ([b59bfa4](https://github.com/nusnewob/kube-changejob/commit/b59bfa4309f87f95af8263e8591ddcfb5814ef85))
- Use resourcepoller, remove lib not used - ([4174b24](https://github.com/nusnewob/kube-changejob/commit/4174b24c084517721e85ae70264b326ed022b57b))
- Basic webhook - ([ef77560](https://github.com/nusnewob/kube-changejob/commit/ef7756080c02636ac5b8c1ddce04c3e6b7d18d13))

## New Contributors ❤️

* @nusnewob made their first contribution
* @dependabot[bot] made their first contribution

<!-- generated by git-cliff -->
