package tagrepo

import (
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

type Tag struct {
	Name string `json:"name"`
}

type TagList struct {
	List []*Tag `json:"list"`
}

type TagMongoDoc struct {
	Name string `bson:"name"`
}

func FindTags() (*TagList, error) {
	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	cursor, err := mongo.TagColl().Find(ctx, bson.M{})

	if err != nil {
		return nil, err
	}

	decodeCtx, cancelDecodeFunc := mongo.DefaultCtx()
	defer cancelDecodeFunc()

	var tagDocs []TagMongoDoc
	err = cursor.All(decodeCtx, &tagDocs)

	if err != nil {
		return nil, err
	}

	var tags []*Tag
	for _, tagDoc := range tagDocs {
		tag := Tag{Name: tagDoc.Name}
		tags = append(tags, &tag)
	}

	var tagList = TagList{List: []*Tag{}}
	if tags != nil {
		tagList.List = tags
	}
	return &tagList, nil
}
