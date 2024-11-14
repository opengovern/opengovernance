import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
    Api,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCreateAwsConnectionRequest,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialResponse,
    GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAwsRequest,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCreateSourceResponse,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCatalogMetrics,
    GithubComKaytuIoKaytuEnginePkgOnboardApiListConnectionSummaryResponse,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCredential,
    GithubComKaytuIoKaytuEnginePkgOnboardApiUpdateCredentialRequest,
    GithubComKaytuIoKaytuEnginePkgOnboardApiConnectionGroup,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCreateConnectionResponse,
    GithubComKaytuIoKaytuEnginePkgOnboardApiChangeConnectionLifecycleStateRequest,
    GithubComKaytuIoKaytuEnginePkgOnboardApiListCredentialResponse,
    GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Request,
    GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Response,
    GithubComKaytuIoKaytuEnginePkgOnboardApiConnectorCount,
    GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialRequest,
    GithubComKaytuIoKaytuEnginePkgOnboardApiConnection,
    GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAzureRequest,
    RequestParams,
} from './api'

import AxiosAPI, { setWorkspace } from './ApiConfig'

interface IuseOnboardApiV1CatalogMetricsListState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCatalogMetrics
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CatalogMetricsList = (
    query?: {
        connector?: ('' | 'AWS' | 'Azure')[]
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CatalogMetricsListState>(
        {
            isLoading: true,
            isExecuted: false,
        }
    )
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqquery:
            | {
                  connector?: ('' | 'AWS' | 'Azure')[]
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CatalogMetricsList(reqquery, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([query, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([query, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, query, params)
    }

    const sendNowWithParams = (
        reqquery:
            | {
                  connector?: ('' | 'AWS' | 'Azure')[]
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectionGroupsListState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiConnectionGroup[]
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectionGroupsList = (
    query?: {
        populateConnections?: boolean
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1ConnectionGroupsListState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqquery:
            | {
                  populateConnections?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectionGroupsList(reqquery, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([query, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([query, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, query, params)
    }

    const sendNowWithParams = (
        reqquery:
            | {
                  populateConnections?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectionGroupsDetailState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiConnectionGroup
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectionGroupsDetail = (
    connectionGroupName: string,
    query?: {
        populateConnections?: boolean
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1ConnectionGroupsDetailState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([connectionGroupName, query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqconnectionGroupName: string,
        reqquery:
            | {
                  populateConnections?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectionGroupsDetail(
                    reqconnectionGroupName,
                    reqquery,
                    reqparamsSignal
                )
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (
        JSON.stringify([connectionGroupName, query, params, autoExecute]) !==
        lastInput
    ) {
        setLastInput(
            JSON.stringify([connectionGroupName, query, params, autoExecute])
        )
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, connectionGroupName, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, connectionGroupName, query, params)
    }

    const sendNowWithParams = (
        reqconnectionGroupName: string,
        reqquery:
            | {
                  populateConnections?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqconnectionGroupName, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectionsAwsCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateConnectionResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectionsAwsCreate = (
    request: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateAwsConnectionRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1ConnectionsAwsCreateState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([request, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateAwsConnectionRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectionsAwsCreate(reqrequest, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([request, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([request, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, request, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, request, params)
    }

    const sendNowWithParams = (
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateAwsConnectionRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqrequest, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectionsSummaryListState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiListConnectionSummaryResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectionsSummaryList = (
    query?: {
        filter?: string

        connector?: ('' | 'AWS' | 'Azure')[]

        connectionId?: string[]

        resourceCollection?: string[]

        connectionGroups?: string[]

        lifecycleState?:
            | 'DISABLED'
            | 'DISCOVERED'
            | 'IN_PROGRESS'
            | 'ONBOARD'
            | 'ARCHIVED'

        healthState?: 'healthy' | 'unhealthy'

        pageSize?: number

        pageNumber?: number

        startTime?: number

        endTime?: number

        needCost?: boolean

        needResourceCount?: boolean

        sortBy?:
            | 'onboard_date'
            | 'resource_count'
            | 'cost'
            | 'growth'
            | 'growth_rate'
            | 'cost_growth'
            | 'cost_growth_rate'
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1ConnectionsSummaryListState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqquery:
            | {
                  filter?: string

                  connector?: ('' | 'AWS' | 'Azure')[]

                  connectionId?: string[]

                  resourceCollection?: string[]

                  connectionGroups?: string[]

                  lifecycleState?:
                      | 'DISABLED'
                      | 'DISCOVERED'
                      | 'IN_PROGRESS'
                      | 'ONBOARD'
                      | 'ARCHIVED'

                  healthState?: 'healthy' | 'unhealthy'

                  pageSize?: number

                  pageNumber?: number

                  startTime?: number

                  endTime?: number

                  needCost?: boolean

                  needResourceCount?: boolean

                  sortBy?:
                      | 'onboard_date'
                      | 'resource_count'
                      | 'cost'
                      | 'growth'
                      | 'growth_rate'
                      | 'cost_growth'
                      | 'cost_growth_rate'
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectionsSummaryList(reqquery, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([query, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([query, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, query, params)
    }

    const sendNowWithParams = (
        reqquery:
            | {
                  filter?: string

                  connector?: ('' | 'AWS' | 'Azure')[]

                  connectionId?: string[]

                  resourceCollection?: string[]

                  connectionGroups?: string[]

                  lifecycleState?:
                      | 'DISABLED'
                      | 'DISCOVERED'
                      | 'IN_PROGRESS'
                      | 'ONBOARD'
                      | 'ARCHIVED'

                  healthState?: 'healthy' | 'unhealthy'

                  pageSize?: number

                  pageNumber?: number

                  startTime?: number

                  endTime?: number

                  needCost?: boolean

                  needResourceCount?: boolean

                  sortBy?:
                      | 'onboard_date'
                      | 'resource_count'
                      | 'cost'
                      | 'growth'
                      | 'growth_rate'
                      | 'cost_growth'
                      | 'cost_growth_rate'
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectionsStateCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: void
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectionsStateCreate = (
    connectionId: string,
    request: GithubComKaytuIoKaytuEnginePkgOnboardApiChangeConnectionLifecycleStateRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1ConnectionsStateCreateState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([connectionId, request, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqconnectionId: string,
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiChangeConnectionLifecycleStateRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectionsStateCreate(
                    reqconnectionId,
                    reqrequest,
                    reqparamsSignal
                )
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (
        JSON.stringify([connectionId, request, params, autoExecute]) !==
        lastInput
    ) {
        setLastInput(
            JSON.stringify([connectionId, request, params, autoExecute])
        )
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, connectionId, request, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, connectionId, request, params)
    }

    const sendNowWithParams = (
        reqconnectionId: string,
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiChangeConnectionLifecycleStateRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqconnectionId, reqrequest, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1ConnectorListState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiConnectorCount[]
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1ConnectorList = (
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1ConnectorListState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1ConnectorList(reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, params)
    }

    const sendNowWithParams = (reqparams: RequestParams) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialListState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiListCredentialResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialList = (
    query?: {
        connector?: '' | 'AWS' | 'Azure'

        health?: 'healthy' | 'unhealthy'

        credentialType?: (
            | 'auto-azure'
            | 'auto-aws'
            | 'manual-aws-org'
            | 'manual-azure-spn'
        )[]

        pageSize?: number

        pageNumber?: number
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CredentialListState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqquery:
            | {
                  connector?: '' | 'AWS' | 'Azure'

                  health?: 'healthy' | 'unhealthy'

                  credentialType?: (
                      | 'auto-azure'
                      | 'auto-aws'
                      | 'manual-aws-org'
                      | 'manual-azure-spn'
                  )[]

                  pageSize?: number

                  pageNumber?: number
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialList(reqquery, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([query, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([query, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, query, params)
    }

    const sendNowWithParams = (
        reqquery:
            | {
                  connector?: '' | 'AWS' | 'Azure'

                  health?: 'healthy' | 'unhealthy'

                  credentialType?: (
                      | 'auto-azure'
                      | 'auto-aws'
                      | 'manual-aws-org'
                      | 'manual-azure-spn'
                  )[]

                  pageSize?: number

                  pageNumber?: number
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialCreate = (
    config: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CredentialCreateState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([config, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialCreate(reqconfig, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([config, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([config, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, config, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, config, params)
    }

    const sendNowWithParams = (
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateCredentialRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqconfig, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialDetailState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCredential
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialDetail = (
    credentialId: string,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CredentialDetailState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([credentialId, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialDetail(reqcredentialId, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([credentialId, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([credentialId, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, credentialId, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, credentialId, params)
    }

    const sendNowWithParams = (
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqcredentialId, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialUpdateState {
    isLoading: boolean
    isExecuted: boolean
    response?: void
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialUpdate = (
    credentialId: string,
    config: GithubComKaytuIoKaytuEnginePkgOnboardApiUpdateCredentialRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CredentialUpdateState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([credentialId, config, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqcredentialId: string,
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiUpdateCredentialRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialUpdate(
                    reqcredentialId,
                    reqconfig,
                    reqparamsSignal
                )
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (
        JSON.stringify([credentialId, config, params, autoExecute]) !==
        lastInput
    ) {
        setLastInput(
            JSON.stringify([credentialId, config, params, autoExecute])
        )
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, credentialId, config, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, credentialId, config, params)
    }

    const sendNowWithParams = (
        reqcredentialId: string,
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiUpdateCredentialRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqcredentialId, reqconfig, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialDeleteState {
    isLoading: boolean
    isExecuted: boolean
    response?: void
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialDelete = (
    credentialId: string,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1CredentialDeleteState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([credentialId, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialDelete(reqcredentialId, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([credentialId, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([credentialId, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, credentialId, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, credentialId, params)
    }

    const sendNowWithParams = (
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqcredentialId, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1CredentialAutoonboardCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiConnection[]
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1CredentialAutoonboardCreate = (
    credentialId: string,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1CredentialAutoonboardCreateState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([credentialId, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1CredentialAutoonboardCreate(
                    reqcredentialId,
                    reqparamsSignal
                )
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([credentialId, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([credentialId, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, credentialId, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, credentialId, params)
    }

    const sendNowWithParams = (
        reqcredentialId: string,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqcredentialId, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1SourceAwsCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateSourceResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1SourceAwsCreate = (
    request: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAwsRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1SourceAwsCreateState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([request, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAwsRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1SourceAwsCreate(reqrequest, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([request, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([request, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, request, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, request, params)
    }

    const sendNowWithParams = (
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAwsRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqrequest, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1SourceAzureCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiCreateSourceResponse
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1SourceAzureCreate = (
    request: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAzureRequest,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1SourceAzureCreateState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([request, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAzureRequest,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1SourceAzureCreate(reqrequest, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([request, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([request, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, request, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, request, params)
    }

    const sendNowWithParams = (
        reqrequest: GithubComKaytuIoKaytuEnginePkgOnboardApiSourceAzureRequest,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqrequest, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1SourceDeleteState {
    isLoading: boolean
    isExecuted: boolean
    response?: void
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1SourceDelete = (
    sourceId: string,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV1SourceDeleteState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([sourceId, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqsourceId: string,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1SourceDelete(reqsourceId, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([sourceId, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([sourceId, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, sourceId, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, sourceId, params)
    }

    const sendNowWithParams = (
        reqsourceId: string,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqsourceId, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV1SourceHealthcheckDetailState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiConnection
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV1SourceHealthcheckDetail = (
    sourceId: string,
    query?: {
        updateMetadata?: boolean
    },
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] =
        useState<IuseOnboardApiV1SourceHealthcheckDetailState>({
            isLoading: true,
            isExecuted: false,
        })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([sourceId, query, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqsourceId: string,
        reqquery:
            | {
                  updateMetadata?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV1SourceHealthcheckDetail(
                    reqsourceId,
                    reqquery,
                    reqparamsSignal
                )
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([sourceId, query, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([sourceId, query, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, sourceId, query, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, sourceId, query, params)
    }

    const sendNowWithParams = (
        reqsourceId: string,
        reqquery:
            | {
                  updateMetadata?: boolean
              }
            | undefined,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqsourceId, reqquery, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}

interface IuseOnboardApiV2CredentialCreateState {
    isLoading: boolean
    isExecuted: boolean
    response?: GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Response
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    error?: any
}

/**
 * URL:
 */
export const useOnboardApiV2CredentialCreate = (
    config: GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Request,
    params: RequestParams = {},
    autoExecute = true,
    overwriteWorkspace: string | undefined = undefined
) => {
    const workspace = useParams<{ ws: string }>().ws
    const [controller, setController] = useState(new AbortController())

    const api = new Api()
    api.instance = AxiosAPI

    const [state, setState] = useState<IuseOnboardApiV2CredentialCreateState>({
        isLoading: true,
        isExecuted: false,
    })
    const [lastInput, setLastInput] = useState<string>(
        JSON.stringify([config, params, autoExecute])
    )

    const sendRequest = (
        abortCtrl: AbortController,
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Request,
        reqparams: RequestParams
    ) => {
        if (!api.instance.defaults.headers.common.Authorization) {
            return
        }

        setState({
            ...state,
            error: undefined,
            isLoading: true,
            isExecuted: true,
        })
        try {
            if (overwriteWorkspace) {
                setWorkspace(overwriteWorkspace)
            } else if (workspace !== undefined && workspace.length > 0) {
                setWorkspace(workspace)
            } else {
                setWorkspace('main')
            }

            const reqparamsSignal = { ...reqparams, signal: abortCtrl.signal }
            api.onboard
                .apiV2CredentialCreate(reqconfig, reqparamsSignal)
                .then((resp) => {
                    setState({
                        ...state,
                        error: undefined,
                        response: resp.data,
                        isLoading: false,
                        isExecuted: true,
                    })
                })
                .catch((err) => {
                    if (
                        err.name === 'AbortError' ||
                        err.name === 'CanceledError'
                    ) {
                        // Request was aborted
                    } else {
                        setState({
                            ...state,
                            error: err,
                            response: undefined,
                            isLoading: false,
                            isExecuted: true,
                        })
                    }
                })
        } catch (err) {
            setState({
                ...state,
                error: err,
                isLoading: false,
                isExecuted: true,
            })
        }
    }

    if (JSON.stringify([config, params, autoExecute]) !== lastInput) {
        setLastInput(JSON.stringify([config, params, autoExecute]))
    }

    useEffect(() => {
        if (autoExecute) {
            controller.abort()
            const newController = new AbortController()
            setController(newController)
            sendRequest(newController, config, params)
        }
    }, [lastInput])

    const { response } = state
    const { isLoading } = state
    const { isExecuted } = state
    const { error } = state
    const sendNow = () => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, config, params)
    }

    const sendNowWithParams = (
        reqconfig: GithubComKaytuIoKaytuEnginePkgOnboardApiV2CreateCredentialV2Request,
        reqparams: RequestParams
    ) => {
        controller.abort()
        const newController = new AbortController()
        setController(newController)
        sendRequest(newController, reqconfig, reqparams)
    }

    return {
        response,
        isLoading,
        isExecuted,
        error,
        sendNow,
        sendNowWithParams,
    }
}
