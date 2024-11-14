import { useNavigate } from 'react-router-dom'
import { useAtom } from 'jotai/index'
import { Fragment, useEffect } from 'react'
import { Popover, Transition } from '@headlessui/react'
import { Card, Flex, Text } from '@tremor/react'
import {
    ArrowTopRightOnSquareIcon,
    ChevronUpDownIcon,
} from '@heroicons/react/24/outline'
import { workspaceAtom } from '../../../../store'
import { useWorkspaceApiV1WorkspacesList } from '../../../../api/workspace.gen'
import {
    capitalizeFirstLetter,
    kebabCaseToLabel,
} from '../../../../utilities/labelMaker'

interface IWorkspaces {
    isCollapsed: boolean
}
export default function Workspaces({ isCollapsed }: IWorkspaces) {
    const navigate = useNavigate()
    const [workspace, setWorkspace] = useAtom(workspaceAtom)
    const url = window.location.pathname.split('/')
    if (url[1] === 'ws') {
        url.shift()
    }
    const wsName = url[1]

    const {
        response: workspaceInfo,
        isExecuted: workspaceInfoExecuted,
        sendNow: sendWorkspaceInfo,
    } = useWorkspaceApiV1WorkspacesList({}, false)

    useEffect(() => {
        if (
            !workspace.current &&
            workspace.list.length < 1 &&
            !workspaceInfoExecuted
        ) {
            sendWorkspaceInfo()
        }
        if (workspace && wsName) {
            if (
                !workspace.current ||
                workspace.list.length < 1 ||
                workspace.current.name !== wsName
            ) {
                const current = workspaceInfo?.filter(
                    (ws) => ws.name === wsName
                )

                setWorkspace({
                    list: workspaceInfo || [],
                    current: current ? current[0] : undefined,
                })
            }
        }
    }, [workspace, workspaceInfo, wsName])

    return (
        <Popover className="relative z-50 border-0 w-full">
            <Popover.Button
                className={`py-3 px-4 w-full rounded-lg cursor-pointer ${
                    isCollapsed ? '!p-1' : 'border border-gray-700'
                }`}
                id="workspace"
                onClick={() =>
                    workspace.list.length === 1
                        ? navigate('/ws/workspaces')
                        : undefined
                }
            >
                <Flex justifyContent={isCollapsed ? 'center' : 'between'}>
                    <Flex className="w-fit gap-2">
                        <Flex
                            justifyContent="center"
                            className={`${
                                isCollapsed ? 'w-9 h-9' : 'w-6 h-6'
                            } rounded-md bg-openg-800`}
                        >
                            <Text className="font-semibold !text-base text-orange-500">
                                {wsName ? wsName[0].toLocaleUpperCase() : 'K'}
                            </Text>
                        </Flex>
                        {!isCollapsed && (
                            <Text className="!text-base text-gray-50">
                                {kebabCaseToLabel(wsName)}
                            </Text>
                        )}
                    </Flex>
                    {!isCollapsed && (
                        <div>
                            {workspace.list.length === 1 ? (
                                <ArrowTopRightOnSquareIcon className="w-5 text-gray-400" />
                            ) : (
                                <ChevronUpDownIcon className="w-5 text-gray-400" />
                            )}
                        </div>
                    )}
                </Flex>
            </Popover.Button>
            <Transition
                as={Fragment}
                enter="transition ease-out duration-200"
                enterFrom="opacity-0 translate-y-1"
                enterTo="opacity-100 translate-y-0"
                leave="transition ease-in duration-150"
                leaveFrom="opacity-100 translate-y-0"
                leaveTo="opacity-0 translate-y-1"
            >
                <Popover.Panel
                    className={`absolute z-10 w-full ${
                        isCollapsed ? 'left-[57px] -top-[4px]' : 'top-full'
                    }`}
                >
                    <Card className="w-full min-w-[256px] bg-openg-950 p-2 mt-2 !ring-gray-600">
                        <div className="w-full pb-2 mb-2 border-b border-b-gray-700">
                            <Text className="mb-2 ml-2">WORKSPACES</Text>
                            {workspace.list
                                .filter((ws) => ws.status === 'PROVISIONED')
                                .map((ws) => (
                                    <Flex
                                        onClick={() =>
                                            navigate(`/ws/${ws.name}`)
                                        }
                                        justifyContent="start"
                                        className={`py-1 px-2 gap-2 my-1 rounded-md cursor-pointer ${
                                            wsName === ws.name
                                                ? 'bg-openg-500 text-gray-200 font-semibold'
                                                : 'text-gray-50 hover:bg-openg-800'
                                        }`}
                                    >
                                        <Flex
                                            justifyContent="center"
                                            className="w-6 rounded-md bg-openg-800"
                                        >
                                            <Text className="font-semibold !text-base text-orange-500">
                                                {capitalizeFirstLetter(
                                                    ws.name && ws.name[0]
                                                        ? ws.name[0]
                                                        : 'K'
                                                )}
                                            </Text>
                                        </Flex>
                                        <Text className="text-inherit">
                                            {ws.name}
                                        </Text>
                                    </Flex>
                                ))}
                        </div>
                        <Flex
                            onClick={() => navigate('/ws/workspaces')}
                            className="p-2 text-gray-300 rounded-md cursor-pointer hover:text-gray-50 hover:bg-openg-800"
                        >
                            <Text className="text-inherit font-semibold">
                                Workspace list
                            </Text>
                            <ArrowTopRightOnSquareIcon className="w-5 text-gray-400" />
                        </Flex>
                    </Card>
                </Popover.Panel>
            </Transition>
        </Popover>
    )
}
