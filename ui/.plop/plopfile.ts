import { appGenerator } from './app-generator'
import { libGenerator } from './lib-generator'
import { NodePlopAPI } from 'plop'
import { copyFiles } from './actions'

module.exports = function (plop: NodePlopAPI) {
  plop.setActionType('copy-files', copyFiles)

  plop.setGenerator('app', appGenerator(plop))
  plop.setGenerator('lib', libGenerator(plop))
}
