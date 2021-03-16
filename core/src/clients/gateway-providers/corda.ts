import axios from 'axios';
import { config } from '../../lib/config';
import { IAPIGatewayAsyncResponse, IAPIGatewaySyncResponse } from '../../lib/interfaces';
import { getLogger } from '../../lib/utils';
const log = getLogger('gateway-providers/corda.ts');

// Asset instance APIs

export const createDescribedAssetInstance = async (assetInstanceID: string, assetDefinitionID: string,
  descriptionHash: string, contentHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedAssetInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetInstanceID: assetInstanceID,
        assetDefinitionID: assetDefinitionID,
        descriptionHash,
        contentHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to create described asset instance ${assetInstanceID}`, err);
  }
};

export const createAssetInstance = async (assetInstanceID: string, assetDefinitionID: string,
  contentHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetInstanceID: assetInstanceID,
        assetDefinitionID: assetDefinitionID,
        contentHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to create asset instance ${assetInstanceID}`, err);
  }
};

export const createAssetInstanceBatch = async (batchHash: string, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createAssetInstanceBatch`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        batchHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to create asset instance batch ${batchHash}`, err);
  }
};

export const setAssetInstanceProperty = async (assetDefinitionID: string, assetInstanceID: string, key: string, value: string,
  participants: string[] | undefined | undefined): Promise<IAPIGatewayAsyncResponse | IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/setAssetInstanceProperty`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        assetDefinitionID: assetDefinitionID,
        assetInstanceID: assetInstanceID,
        key,
        value,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to set asset instance property ${key} (instance=${assetInstanceID})`, err);
  }
};

export const createDescribedPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string, recipient: string, amount: number, descriptionHash: string, participants: string[] | undefined):
  Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createDescribedPaymentInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: paymentInstanceID,
        paymentDefinitionID: paymentDefinitionID,
        recipient,
        amount,
        descriptionHash,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to create described asset payment instance ${paymentInstanceID}`, err);
  }
};

export const createPaymentInstance = async (paymentInstanceID: string, paymentDefinitionID: string,
  recipient: string, amount: number, participants: string[] | undefined): Promise<IAPIGatewaySyncResponse> => {
  try {
    const response = await axios({
      method: 'post',
      url: `${config.apiGateway.apiEndpoint}/createPaymentInstance`,
      auth: {
        username: config.apiGateway.auth?.user ?? config.appCredentials.user,
        password: config.apiGateway.auth?.password ?? config.appCredentials.password
      },
      data: {
        paymentInstanceID: paymentInstanceID,
        paymentDefinitionID: paymentDefinitionID,
        recipient,
        amount,
        participants
      }
    });
    return { ...response.data, type: 'sync' };
  } catch (err) {
    return handleError(`Failed to create asset payment instance ${paymentInstanceID}`, err);
  }
};

function handleError(msg: string, err: any): Promise<IAPIGatewaySyncResponse> {
  const errMsg = err.response?.data?.error ?? err.response.data.message ?? err.toString();
  log.error(`${msg}. ${errMsg}`);
  throw new Error(msg);
}