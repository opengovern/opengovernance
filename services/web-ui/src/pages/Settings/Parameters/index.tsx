import { PlusIcon } from '@heroicons/react/24/outline'
import {
    ArrowRightCircleIcon,
    KeyIcon,
    PlusCircleIcon,
    TrashIcon,
} from '@heroicons/react/24/solid'
import { useEffect, useState } from 'react'
import {
    Button,
    Card,
    Divider,
    Flex,
    TextInput,
    Textarea,
    Title,
} from '@tremor/react'
import { useAtom, useAtomValue } from 'jotai'
import {
    useMetadataApiV1QueryParameterCreate,
    useMetadataApiV1QueryParameterList,
} from '../../../api/metadata.gen'
import { getErrorMessage } from '../../../types/apierror'
import { notificationAtom } from '../../../store'
import { searchAtom, useURLParam } from '../../../utilities/urlstate'
import TopHeader from '../../../components/Layout/Header'

interface IParam {
    key: string
    value: string
}

export default function SettingsParameters() {
    const [notif, setNotif] = useAtom(notificationAtom)

    const [params, setParams] = useState<IParam[]>([])
    const {
        response: parameters,
        isLoading,
        isExecuted,
        sendNow: refresh,
    } = useMetadataApiV1QueryParameterList()

    const {
        isLoading: updateIsLoading,
        isExecuted: updateIsExecuted,
        error: updateError,
        sendNowWithParams,
    } = useMetadataApiV1QueryParameterCreate({}, {}, false)

    useEffect(() => {
        if (!updateIsLoading && updateIsExecuted) {
            const err = getErrorMessage(updateError)
            if (err !== '') {
                setNotif({
                    text: `Failed to update parameters due to: ${err}`,
                    type: 'error',
                    position: 'bottomLeft',
                })
            } else {
                setNotif({
                    text: `Successfully updated`,
                    type: 'success',
                    position: 'bottomLeft',
                })
            }
        }
    }, [updateIsLoading])

    useEffect(() => {
        setParams(
            parameters?.queryParameters?.map((p) => {
                return {
                    key: p.key || '',
                    value: p.value || '',
                }
            }) || []
        )
    }, [parameters])
    const [keyParam] = useURLParam('key', '')

    useEffect(() => {
        if (keyParam.length > 0) {
            const elem = document.getElementById(keyParam)
            elem?.scrollIntoView({
                behavior: 'smooth',
                block: 'center',
                inline: 'center',
            })
        }
    }, [params])

    const updateKey = (newKey: string, idx: number) => {
        setParams(
            params.map((v, i) =>
                i === idx
                    ? {
                          key: newKey,
                          value: v.value,
                      }
                    : v
            )
        )
    }

    const updateValue = (newValue: string, idx: number) => {
        setParams(
            params.map((v, i) =>
                i === idx
                    ? {
                          key: v.key,
                          value: newValue.trim(),
                      }
                    : v
            )
        )
    }

    const deleteRow = (idx: number) => {
        setParams(params.filter((v, i) => i !== idx))
    }

    const addRow = () => {
        setParams([...params, { key: '', value: '' }])
    }

    return (
        <>
            {/* <TopHeader /> */}
            <Card key="summary" className="">
                <Flex>
                    <Title className="font-semibold">Variables</Title>
                    <Button
                        variant="secondary"
                        icon={PlusIcon}
                        onClick={addRow}
                    >
                        Add
                    </Button>
                </Flex>
                <Divider />

                <Flex flexDirection="col" className="mt-4">
                    {params.map((p, idx) => {
                        return (
                            <Flex flexDirection="row" className="mb-4">
                                <KeyIcon className="w-10 mr-3" />
                                <TextInput
                                    id={p.key}
                                    value={p.key}
                                    onValueChange={(e) =>
                                        updateKey(String(e), idx)
                                    }
                                    className={
                                        keyParam === p.key
                                            ? 'border-red-500'
                                            : ''
                                    }
                                />
                                <ArrowRightCircleIcon className="w-10 mx-3" />
                                <Textarea
                                    value={p.value}
                                    onValueChange={(e) =>
                                        updateValue(String(e), idx)
                                    }
                                    rows={1}
                                    className={
                                        keyParam === p.key
                                            ? 'border-red-500'
                                            : ''
                                    }
                                />
                                <TrashIcon
                                    className="w-10 ml-3 hover:cursor-pointer"
                                    onClick={() => deleteRow(idx)}
                                />
                            </Flex>
                        )
                    })}
                </Flex>
                <Flex flexDirection="row" justifyContent="end">
                    <Button
                        variant="secondary"
                        className="mx-4"
                        onClick={() => {
                            refresh()
                        }}
                        loading={isExecuted && isLoading}
                    >
                        Reset
                    </Button>
                    <Button
                        onClick={() => {
                            sendNowWithParams(
                                {
                                    queryParameters: params,
                                },
                                {}
                            )
                        }}
                        loading={updateIsExecuted && updateIsLoading}
                    >
                        Save
                    </Button>
                </Flex>
            </Card>
        </>
    )
}
