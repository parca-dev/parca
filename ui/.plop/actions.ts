import { ActionConfig, NodePlopAPI } from 'plop'
import globby from 'globby'
import fse from 'fs-extra'
import replaceInFiles from 'replace-in-files'

export const copyFiles = async (answers: object, config: ActionConfig, plop: NodePlopAPI) => {
  const configData = config.data as any

  const allFiles = await globby([configData.source], {
    gitignore: true,
    dot: true
  })
  for (let fileName of allFiles) {
    const destFileName = fileName.replace(configData.source, configData.dest)
    console.log(`- ${destFileName}`)
    await fse.copy(fileName, destFileName)
  }

  for (let key in configData.replaceInFiles) {
    await replaceInFiles({
      files: [`${configData.dest}/**/*`, `${configData.dest}/*`],
      from: new RegExp(key, 'g'),
      to: configData.replaceInFiles[key]
    })
  }

  return await Promise.resolve('success')
}
