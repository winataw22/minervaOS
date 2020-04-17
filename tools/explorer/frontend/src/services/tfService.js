import axios from 'axios'
import config from '../../public/config'

export default {
  getName () {
    return axios.post(`/${window.config.identityActor}/threebot_name`, {
      args: {}
    })
  },
  getUser (name) {
    return axios.get(`${config.tfApiUrl}/users`, {
      params: {
        name: name
      }
    })
  },
  getFarms (userId) {
    return axios.get(`${config.tfApiUrl}/farms`, {
      params: {
        threebot_id: userId
      }
    })
  },
  registerFarm (farm) {
    return axios.post(`${config.tfApiUrl}/farms/register`,
      {
        farm: farm
      }
    )
  },
  updateFarm (farmId, farm) {
    return axios.post(`${config.tfApiUrl}/farms/update`, {
      args: {
        farm_id: farmId,
        farm: farm
      }
    })
  },
  getNodes (farm_id = undefined, size, page) {
    return axios.get(`${config.tfApiUrl}/nodes`, {
      params: {
        farm_id: farm_id,
        size,
        page
      }
    })
  },
  registeredfarms (size, page) {
    return axios.get(`${config.tfApiUrl}/farms`, {
      params: {
        size,
        page
      }
    })
  },
  news () {
    return axios.get(`${config.tfApiUrl}/news`)
  },
  getExplorerConstants () {
    return axios.get(`${config.tfExplorerUrl}`)
  },
  getExplorerBlockByHeight (height) {
    return axios.get(`${config.tfExplorerUrl}/blocks/${height}`)
  }
}
