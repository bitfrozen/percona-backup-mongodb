package pbm

import (
	"encoding/json"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

func (p *PBM) ResyncBackupList() error {
	stg, err := p.GetStorage()
	if err != nil {
		return errors.Wrap(err, "unable to get backup store")
	}

	bcps, err := stg.Files(MetadataFileSuffix)
	if err != nil {
		return errors.Wrap(err, "get a backups list from the storage")
	}

	err = p.archiveBackupsMeta(BcpCollection, BcpOldCollection)
	if err != nil {
		return errors.Wrapf(err, "copy current backups meta from %s to %s", BcpCollection, BcpOldCollection)
	}

	_, err = p.Conn.Database(DB).Collection(BcpCollection).DeleteMany(p.ctx, bson.M{})
	if err != nil {
		return errors.Wrap(err, "remove current backups meta")
	}

	if len(bcps) == 0 {
		return nil
	}

	var ins []interface{}
	for _, b := range bcps {
		v := BackupMeta{}
		err = json.Unmarshal(b, &v)
		if err != nil {
			return errors.Wrap(err, "unmarshal backup meta")
		}
		ins = append(ins, v)
	}
	_, err = p.Conn.Database(DB).Collection(BcpCollection).InsertMany(p.ctx, ins)
	if err != nil {
		return errors.Wrap(err, "insert retrieved backups meta")
	}

	return nil
}

func (p *PBM) archiveBackupsMeta(from, to string) error {
	err := p.Conn.Database(DB).Collection(to).Drop(p.ctx)
	if err != nil {
		return errors.Wrap(err, "failed to remove old archive from backups metadata")
	}

	cur, err := p.Conn.Database(DB).Collection(from).Find(p.ctx, bson.M{})
	if err != nil {
		return errors.Wrap(err, "get current backups meta")
	}
	for cur.Next(p.ctx) {
		_, err = p.Conn.Database(DB).Collection(to).InsertOne(p.ctx, cur.Current)
		if err != nil {
			return errors.Wrapf(err, "insert")
		}

	}

	return cur.Err()
}
