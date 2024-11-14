import { Fragment, useState } from 'react'
import { Popover, Transition } from '@headlessui/react'
import {
    CommandLineIcon,
    DocumentDuplicateIcon,
} from '@heroicons/react/24/outline'
import { Button, Card, Flex, Text } from '@tremor/react'
import clipboardCopy from 'clipboard-copy'
import { ReactComponent as AppleIcon } from '../../../../../icons/Apple.svg'
import { ReactComponent as LinuxIcon } from '../../../../../icons/Vector.svg'
import { ReactComponent as WindowsIcon } from '../../../../../icons/windows-174-svgrepo-com 1.svg'

const tabs = [
    {
        name: 'macOS',
        href: 'https://github.com/kaytu-io/cli/releases',
        icon: <AppleIcon className="w-6 h-6 m-1" />,
        commands: (
            <div>
                $ brew tap kaytu-io/cli-tap <br /> $ brew install kaytu
            </div>
        ),
        clipboard: 'brew tap kaytu-io/cli-tap && brew install kaytu',
    },
    {
        name: 'Linux',
        href: 'https://github.com/kaytu-io/cli/releases',
        icon: <LinuxIcon className="w-5 h-5 m-1" />,
        commands:
            '$ wget -qO - https://raw.githubusercontent.com/kaytu-io/cli/main/scripts/install.sh | bash',
        clipboard:
            'wget -qO - https://raw.githubusercontent.com/kaytu-io/cli/main/scripts/install.sh | bash',
    },
    {
        name: 'Windows',
        href: 'https://github.com/kaytu-io/cli/releases',
        icon: <WindowsIcon className="w-6 h-6 m-1" />,
    },
]

function classNames(...classes: any) {
    return classes.filter(Boolean).join(' ')
}

export function CLITabs() {
    const [currentTab, setCurrentTab] = useState<number>(0)
    const [showCopied, setShowCopied] = useState<boolean>(false)

    const getCurrentTab = () => {
        return tabs.at(currentTab)
    }

    return (
        <>
            <div>
                <nav className="isolate flex divide-x divide-gray-200 dark:divide-gray-700 rounded-lg shadow">
                    {tabs.map((tab, tabIdx) => (
                        <Flex
                            flexDirection="row"
                            justifyContent="center"
                            className={classNames(
                                currentTab === tabIdx
                                    ? 'bg-openg-50 dark:bg-openg-950 text-openg-800 dark:text-white fill-blue-600'
                                    : 'bg-gray-50 dark:bg-gray-950 text-gray-600 dark:text-white fill-gray-600 hover:text-gray-700',
                                tabIdx === 0 ? 'rounded-l-lg' : '',
                                tabIdx === tabs.length - 1
                                    ? 'rounded-r-lg'
                                    : '',
                                'group cursor-pointer relative min-w-0 flex-1 overflow-hidden py-4 px-4 text-center text-sm font-medium focus:z-10'
                            )}
                            onClick={() => setCurrentTab(tabIdx)}
                        >
                            {tab.icon}
                            <span>{tab.name}</span>
                            <span
                                aria-hidden="true"
                                className={classNames(
                                    'bg-transparent',
                                    'absolute inset-x-0 bottom-0 h-0.5'
                                )}
                            />
                        </Flex>
                    ))}
                </nav>
            </div>
            <Flex flexDirection="col" justifyContent="start">
                <a
                    href={getCurrentTab()?.href}
                    target="_blank"
                    rel="noreferrer"
                >
                    <Button className="my-8 bg-openg-600">
                        Download for {getCurrentTab()?.name}
                    </Button>
                </a>

                {getCurrentTab()?.commands && (
                    <Card
                        className="w-3/4 text-gray-800 font-mono cursor-pointer p-2.5 !ring-gray-600"
                        onClick={() => {
                            setShowCopied(true)
                            setTimeout(() => {
                                setShowCopied(false)
                            }, 2000)
                            clipboardCopy(getCurrentTab()?.clipboard || '')
                        }}
                    >
                        <Flex flexDirection="row">
                            <Text className="px-1.5 text-gray-800 truncate">
                                {getCurrentTab()?.commands}
                            </Text>
                            <Flex flexDirection="col" className="h-5 w-5">
                                <DocumentDuplicateIcon className="h-5 w-5 text-openg-600 cursor-pointer" />
                                <Text
                                    className={`${
                                        showCopied ? '' : 'hidden'
                                    } absolute -bottom-4 bg-openg-600 text-white rounded-md p-1`}
                                >
                                    Copied!
                                </Text>
                            </Flex>
                        </Flex>
                    </Card>
                )}

                <Text className="mt-3 mb-8 text-gray-400" />
            </Flex>
        </>
    )
}

interface ICLIMenu {
    isCollapsed: boolean
}

export default function CLIMenu({ isCollapsed }: ICLIMenu) {
    return (
        <Popover className="relative z-50 border-0 w-full">
            <Popover.Button
                className={`w-full px-6 py-2 flex items-center rounded-md gap-2.5 text-gray-50 hover:bg-openg-800 ${
                    isCollapsed ? '!p-2' : ''
                }`}
                id="CLI"
            >
                <CommandLineIcon className="h-5 w-5 stroke-2 text-gray-400" />
                {!isCollapsed && <Text className="text-inherit">CLI</Text>}
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
                    className={`absolute ${
                        isCollapsed ? 'left-[57px]' : 'left-[292px]'
                    } bottom-0 z-10`}
                >
                    <Card className="p-0 dark:!ring-gray-600 w-96">
                        <CLITabs />
                    </Card>
                </Popover.Panel>
            </Transition>
        </Popover>
    )
}
