import { useState } from 'react'
import { DocumentDuplicateIcon } from '@heroicons/react/24/outline'
import { Card, Flex, Text } from '@tremor/react'
import clipboardCopy from 'clipboard-copy'
import Spinner from '../Spinner'

interface ICodeBlock {
    command: string
    className?: string
    text?: string
    truncate?: boolean
    loading?: boolean
}

export function CodeBlock({
    command,
    text,
    className,
    truncate,
    loading,
}: ICodeBlock) {
    const [showCopied, setShowCopied] = useState<boolean>(false)
    return (
        <Card
            className={`w-full text-gray-800 font-mono cursor-pointer p-2.5 ${className}`}
            onClick={() => {
                if (!loading) {
                    setShowCopied(true)
                    setTimeout(() => {
                        setShowCopied(false)
                    }, 2000)
                    clipboardCopy(command)
                }
            }}
        >
            <Flex flexDirection="row">
                {loading ? (
                    <Spinner />
                ) : (
                    <Text
                        className={`px-1.5 text-gray-800 ${
                            truncate ? 'truncate' : ''
                        }`}
                    >
                        {text || command}
                    </Text>
                )}

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
    )
}
