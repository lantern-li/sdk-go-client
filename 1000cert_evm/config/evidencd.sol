pragma solidity 0.4.19;

contract Token {

    mapping(string => string) private certMap;

    function setCert(string memory hash, string memory cert) public returns(string) {
        certMap[hash]=cert;
        return hash;
    }

    function getCert(string memory hash) public returns(string) {
        return certMap[hash];
    }
}