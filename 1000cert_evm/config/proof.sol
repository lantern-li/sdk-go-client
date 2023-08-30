// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;


contract Proof {
    // todo add business unique id
    struct Record {
        string fileId;
        string fileHash;
        uint256 timestamp;
        bool exists;
    }

    mapping(string => Record) private records;

    event NewRecord(string fileId, string fileHash, uint256 timestamp);

    //todo 是否需要加白名单
    function addRecord(string memory _fileId, string memory _fileHash) public {
        require(!records[_fileId].exists, "File ID already exists");
        records[_fileId] = Record(_fileId, _fileHash, block.timestamp, true);
        emit NewRecord(_fileId, _fileHash, block.timestamp);
    }

    function getRecord(string memory _fileId) public view returns ( string memory,  string memory, uint256) {
        Record memory record = records[_fileId];
        require(records[_fileId].exists, "File ID not found");
        return (record.fileId, record.fileHash, record.timestamp);
    }
}
